// Package tlsx performs inspection TLS handshakes (optionally after STARTTLS) against a pinned
// address and summarizes certificates.
package tlsx

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/tools/rttstats"
)

// StartTLS protocols supported before the handshake.
var StartTLSProtocols = []string{"", "smtp", "imap", "pop3", "ftp", "xmpp"}

// RootsForTests replaces the system trust store when set.
var RootsForTests *x509.CertPool

// Cert summarizes one certificate.
type Cert struct {
	Subject      string   `json:"subject"`
	Issuer       string   `json:"issuer"`
	Serial       string   `json:"serial"`
	NotBefore    string   `json:"notBefore"`
	NotAfter     string   `json:"notAfter"`
	DaysLeft     int      `json:"daysLeft"`
	DNSNames     []string `json:"dnsNames"`
	IPAddresses  []string `json:"ipAddresses"`
	KeyAlgorithm string   `json:"keyAlgorithm"`
	SigAlgorithm string   `json:"signatureAlgorithm"`
	IsCA         bool     `json:"isCA"`
	SHA256       string   `json:"sha256"`
	SelfSigned   bool     `json:"selfSigned"`
}

// Session describes a completed handshake.
type Session struct {
	Version     string  `json:"version"`
	Cipher      string  `json:"cipher"`
	ALPN        string  `json:"alpn"`
	ServerName  string  `json:"serverName"`
	OCSPStapled bool    `json:"ocspStapled"`
	Chain       []Cert  `json:"chain"`
	Verified    bool    `json:"verified"`
	VerifyError string  `json:"verifyError"`
	HandshakeMs float64 `json:"handshakeMs"`
	StartTLS    string  `json:"starttls"`
	ConnectedTo string  `json:"connectedTo"`
	// Certs is the presented chain (leaf first) and OCSPResponse the stapled response, for
	// tools that need the raw data (revocation checks); neither is serialized.
	Certs        []*x509.Certificate `json:"-"`
	OCSPResponse []byte              `json:"-"`
}

// Options for Handshake.
type Options struct {
	ServerName string
	StartTLS   string
	ALPN       []string
	Timeout    time.Duration
	// Now is used for DaysLeft (tests).
	Now time.Time
}

// Handshake dials addr (a pinned ip:port), runs STARTTLS if asked, then TLS with SNI and
// returns the session summary. Certificate problems are reported, not fatal.
func Handshake(ctx context.Context, addr string, o Options) (*Session, error) {
	if o.Timeout <= 0 {
		o.Timeout = 10 * time.Second
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	d := net.Dialer{}
	start := time.Now()
	raw, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer raw.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = raw.SetDeadline(dl)
	}
	if o.StartTLS != "" {
		if err := startTLS(raw, o.StartTLS, o.ServerName); err != nil {
			return nil, fmt.Errorf("starttls: %w", err)
		}
	}
	sni := o.ServerName
	cfg := &tls.Config{ServerName: sni, NextProtos: o.ALPN, InsecureSkipVerify: true, MinVersion: tls.VersionTLS10} //nolint:gosec // inspection: verified below
	conn := tls.Client(raw, cfg)
	if err := conn.HandshakeContext(ctx); err != nil {
		return nil, fmt.Errorf("handshake: %w", err)
	}
	st := conn.ConnectionState()
	s := &Session{
		Version: tls.VersionName(st.Version), Cipher: tls.CipherSuiteName(st.CipherSuite), ALPN: st.NegotiatedProtocol,
		ServerName: sni, OCSPStapled: len(st.OCSPResponse) > 0, HandshakeMs: rttstats.Ms(time.Since(start)),
		StartTLS: o.StartTLS, ConnectedTo: addr, Chain: []Cert{},
	}
	s.Certs, s.OCSPResponse = st.PeerCertificates, st.OCSPResponse
	for _, c := range st.PeerCertificates {
		s.Chain = append(s.Chain, Summarize(c, o.Now))
	}
	if err := Verify(st.PeerCertificates, sni, o.Now); err != nil {
		s.VerifyError = err.Error()
	} else {
		s.Verified = true
	}
	return s, nil
}

// Verify checks the chain for name (an IP literal verifies against IP SANs).
func Verify(chain []*x509.Certificate, name string, now time.Time) error {
	if len(chain) == 0 {
		return errors.New("no certificate presented")
	}
	inter := x509.NewCertPool()
	for _, c := range chain[1:] {
		inter.AddCert(c)
	}
	_, err := chain[0].Verify(x509.VerifyOptions{DNSName: name, Intermediates: inter, Roots: RootsForTests, CurrentTime: now})
	if err == nil {
		return nil
	}
	var hn x509.HostnameError
	var ua x509.UnknownAuthorityError
	var inv x509.CertificateInvalidError
	switch {
	case errors.As(err, &hn):
		return fmt.Errorf("certificate is not valid for %s", name)
	case errors.As(err, &ua):
		return errors.New("certificate chain is not trusted (unknown authority or missing intermediate)")
	case errors.As(err, &inv) && inv.Reason == x509.Expired:
		return errors.New("certificate has expired or is not yet valid")
	}
	return fmt.Errorf("verification failed: %w", err)
}

// Summarize describes one certificate.
func Summarize(c *x509.Certificate, now time.Time) Cert {
	fp := sha256.Sum256(c.Raw)
	out := Cert{
		Subject: c.Subject.String(), Issuer: c.Issuer.String(), Serial: strings.ToUpper(c.SerialNumber.Text(16)),
		NotBefore: c.NotBefore.UTC().Format(time.RFC3339), NotAfter: c.NotAfter.UTC().Format(time.RFC3339),
		DaysLeft: int(c.NotAfter.Sub(now).Hours() / 24), DNSNames: c.DNSNames, IPAddresses: []string{},
		KeyAlgorithm: keyDesc(c), SigAlgorithm: c.SignatureAlgorithm.String(), IsCA: c.IsCA,
		SHA256: colonHex(fp[:]), SelfSigned: c.Subject.String() == c.Issuer.String() && c.CheckSignatureFrom(c) == nil,
	}
	if out.DNSNames == nil {
		out.DNSNames = []string{}
	}
	for _, ip := range c.IPAddresses {
		out.IPAddresses = append(out.IPAddresses, ip.String())
	}
	return out
}

func keyDesc(c *x509.Certificate) string {
	switch k := c.PublicKey.(type) {
	case *rsa.PublicKey:
		return fmt.Sprintf("RSA %d bit", k.N.BitLen())
	case *ecdsa.PublicKey:
		return "ECDSA " + k.Curve.Params().Name
	case ed25519.PublicKey:
		return "Ed25519"
	}
	return c.PublicKeyAlgorithm.String()
}

func colonHex(b []byte) string {
	parts := make([]string, len(b))
	for i, x := range b {
		parts[i] = fmt.Sprintf("%02X", x)
	}
	return strings.Join(parts, ":")
}

// maxPlaintext bounds what a server may send before the TLS upgrade.
const maxPlaintext = 64 << 10

// startTLS runs the plaintext upgrade dialog for proto. domain is the XMPP stream target.
func startTLS(c net.Conn, proto, domain string) error {
	r := bufio.NewReader(io.LimitReader(c, maxPlaintext))
	readLine := func() (string, error) {
		l, err := r.ReadString('\n')
		return strings.TrimRight(l, "\r\n"), err //nolint:wrapcheck // wrapped by caller
	}
	// SMTP/FTP multi-line replies: "250-..." continues, "250 ..." ends.
	readReply := func(code string) error {
		for {
			l, err := readLine()
			if err != nil {
				return err
			}
			if !strings.HasPrefix(l, code) {
				return fmt.Errorf("unexpected reply %q", truncate(l))
			}
			if len(l) < 4 || l[3] != '-' {
				return nil
			}
		}
	}
	send := func(s string) error {
		_, err := fmt.Fprintf(c, "%s\r\n", s)
		return err //nolint:wrapcheck // wrapped by caller
	}
	switch proto {
	case "smtp":
		if err := readReply("220"); err != nil {
			return err
		}
		if err := send("EHLO nettools.invalid"); err != nil {
			return err
		}
		if err := readReply("250"); err != nil {
			return err
		}
		if err := send("STARTTLS"); err != nil {
			return err
		}
		return readReply("220")
	case "ftp":
		if err := readReply("220"); err != nil {
			return err
		}
		if err := send("AUTH TLS"); err != nil {
			return err
		}
		return readReply("234")
	case "pop3":
		if l, err := readLine(); err != nil || !strings.HasPrefix(l, "+OK") {
			return fmt.Errorf("unexpected greeting: %w", errors.Join(err, errors.New(truncate(l))))
		}
		if err := send("STLS"); err != nil {
			return err
		}
		if l, err := readLine(); err != nil || !strings.HasPrefix(l, "+OK") {
			return fmt.Errorf("STLS refused: %s", truncate(l))
		}
		return nil
	case "imap":
		if l, err := readLine(); err != nil || !strings.HasPrefix(l, "* OK") {
			return fmt.Errorf("unexpected greeting: %s", truncate(l))
		}
		if err := send("a1 STARTTLS"); err != nil {
			return err
		}
		for {
			l, err := readLine()
			if err != nil {
				return err
			}
			if strings.HasPrefix(l, "a1 ") {
				if strings.HasPrefix(l, "a1 OK") {
					return nil
				}
				return fmt.Errorf("STARTTLS refused: %s", truncate(l))
			}
		}
	case "xmpp":
		// RFC 6120 §5: open a client stream, expect <starttls/> in the features, then <proceed/>.
		if err := send(fmt.Sprintf("<?xml version='1.0'?><stream:stream to='%s' xmlns='jabber:client' xmlns:stream='http://etherx.jabber.org/streams' version='1.0'>", xmlEscape(domain))); err != nil {
			return err
		}
		features, err := readUntil(r, "</stream:features>")
		if err != nil {
			return err
		}
		if !strings.Contains(features, "urn:ietf:params:xml:ns:xmpp-tls") {
			return errors.New("the server does not offer STARTTLS")
		}
		if err := send("<starttls xmlns='urn:ietf:params:xml:ns:xmpp-tls'/>"); err != nil {
			return err
		}
		reply, err := readUntil(r, "/>")
		if err != nil {
			return err
		}
		if !strings.Contains(reply, "<proceed") {
			return fmt.Errorf("STARTTLS refused: %s", truncate(reply))
		}
		return nil
	}
	return fmt.Errorf("unsupported protocol %q", proto)
}

// readUntil reads XML until marker appears (bounded by maxPlaintext via the reader).
func readUntil(r *bufio.Reader, marker string) (string, error) {
	var b strings.Builder
	for !strings.Contains(b.String(), marker) {
		chunk, err := r.ReadString('>')
		b.WriteString(chunk)
		if err != nil {
			return b.String(), fmt.Errorf("read: %w", err)
		}
	}
	return b.String(), nil
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func truncate(s string) string {
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
