package tlsinspect

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/netx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tlsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
)

// Limits for TLS-13 (chain validator).
const (
	MaxChainCerts  = 12
	MaxAIAFetches  = 4
	MaxAIABytes    = 64 << 10
	MaxChainPEMLen = 128 << 10
)

// ChainTool is TLS-13: validate a pasted certificate chain — order, signatures, missing
// intermediates (fetched via AIA) and trust against the system roots.
type ChainTool struct {
	deps tools.Deps
	// Fetch overrides the AIA download (tests).
	Fetch func(ctx context.Context, url string) ([]byte, error)
	// Now overrides the clock (tests).
	Now func() time.Time
}

func (*ChainTool) ID() string                                { return "tls.chain" }
func (*ChainTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *ChainTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Fetch == nil {
		client := netx.PolicyClient(d.Target, d.UserAgent, 8*time.Second)
		t.Fetch = func(ctx context.Context, url string) ([]byte, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, err //nolint:wrapcheck // reported per certificate
			}
			resp, err := client.Do(req)
			if err != nil {
				return nil, err //nolint:wrapcheck // reported per certificate
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return nil, errors.New(resp.Status)
			}
			return io.ReadAll(io.LimitReader(resp.Body, MaxAIABytes)) //nolint:wrapcheck // reported per certificate
		}
	}
	if t.Now == nil {
		t.Now = time.Now
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Post("/tls/chain-validate", t.handle)
}

// ChainCert is one certificate in the result.
type ChainCert struct {
	tlsx.Cert
	// Role: leaf, intermediate, root or extra (not on the path).
	Role string `json:"role"`
	// Source: input or aia (fetched from the issuer URL of the previous certificate).
	Source string `json:"source"`
	// InputIndex is the position in the pasted text (-1 for fetched certificates).
	InputIndex int `json:"inputIndex"`
	// SignatureOK reports whether the next certificate on the path signed this one.
	SignatureOK *bool  `json:"signatureOk"`
	Expired     bool   `json:"expired"`
	AIA         string `json:"aia"`
}

// ChainResult is the TLS-13 payload.
type ChainResult struct {
	Certs []ChainCert `json:"certs"`
	// Path lists indices into Certs from the leaf upwards.
	Path []int `json:"path"`
	// Ordered: the pasted certificates already follow the path order (leaf first).
	Ordered bool `json:"ordered"`
	// Complete: the pasted certificates alone verify (no intermediate had to be fetched).
	Complete    bool     `json:"complete"`
	Trusted     bool     `json:"trusted"`
	VerifyError string   `json:"verifyError"`
	Host        string   `json:"host"`
	HostOK      *bool    `json:"hostOk"`
	Notes       []string `json:"notes"`
}

// issuedBy reports whether parent issued child (names match and the signature verifies).
func issuedBy(child, parent *x509.Certificate) bool {
	return bytes.Equal(child.RawIssuer, parent.RawSubject) && child.CheckSignatureFrom(parent) == nil
}

func selfSigned(c *x509.Certificate) bool { return issuedBy(c, c) }

// ParsePEMCerts returns the CERTIFICATE blocks of a PEM text.
func ParsePEMCerts(text string) ([]*x509.Certificate, error) {
	var out []*x509.Certificate
	rest := []byte(text)
	for {
		var b *pem.Block
		b, rest = pem.Decode(rest)
		if b == nil {
			break
		}
		if b.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(b.Bytes)
		if err != nil {
			return nil, errors.New("a CERTIFICATE block could not be parsed")
		}
		out = append(out, c)
	}
	return out, nil
}

// ParseAIA decodes a caIssuers download: DER, PEM or a certs-only PKCS#7 (.p7c).
func ParseAIA(b []byte) ([]*x509.Certificate, error) {
	if c, err := x509.ParseCertificate(b); err == nil {
		return []*x509.Certificate{c}, nil
	}
	if cs, err := ParsePEMCerts(string(b)); err == nil && len(cs) > 0 {
		return cs, nil
	}
	var ci struct {
		Type    asn1.ObjectIdentifier
		Content asn1.RawValue `asn1:"explicit,tag:0"`
	}
	if _, err := asn1.Unmarshal(b, &ci); err != nil {
		return nil, errors.New("not a certificate or PKCS#7 bundle")
	}
	var sd struct {
		Version          int
		DigestAlgorithms asn1.RawValue
		ContentInfo      asn1.RawValue
		Certificates     asn1.RawValue `asn1:"optional,tag:0"`
	}
	if _, err := asn1.Unmarshal(ci.Content.Bytes, &sd); err != nil || len(sd.Certificates.Bytes) == 0 {
		return nil, errors.New("PKCS#7 bundle without certificates")
	}
	return x509.ParseCertificates(sd.Certificates.Bytes) //nolint:wrapcheck // message shown as is
}

// Analyze orders the chain, fetches missing issuers and verifies trust.
func (t *ChainTool) Analyze(ctx context.Context, input []*x509.Certificate, host string) ChainResult {
	now := t.Now()
	res := ChainResult{Certs: []ChainCert{}, Path: []int{}, Notes: []string{}, Host: host}
	all := slices.Clone(input)
	source := make([]string, len(all))
	for i := range source {
		source[i] = "input"
	}
	// Leaf: a certificate that issued none of the others (prefer the first such one).
	leaf := 0
	for i, c := range all {
		issuer := false
		for j, o := range all {
			if i != j && issuedBy(o, c) {
				issuer = true
				break
			}
		}
		if !issuer {
			leaf = i
			break
		}
	}
	path := []int{leaf}
	for len(path) <= MaxChainCerts {
		cur := all[path[len(path)-1]]
		if selfSigned(cur) {
			break
		}
		next := -1
		for j, o := range all {
			if !slices.Contains(path, j) && issuedBy(cur, o) {
				next = j
				break
			}
		}
		if next < 0 && len(all) < MaxChainCerts+MaxAIAFetches {
			for _, u := range cur.IssuingCertificateURL {
				b, err := t.Fetch(ctx, u)
				if err != nil {
					res.Notes = append(res.Notes, "AIA fetch failed: "+u)
					continue
				}
				cs, err := ParseAIA(b)
				if err != nil {
					res.Notes = append(res.Notes, "AIA download is not a certificate: "+u)
					continue
				}
				for _, c := range cs {
					if issuedBy(cur, c) {
						all, source = append(all, c), append(source, "aia")
						next = len(all) - 1
						break
					}
				}
				if next >= 0 {
					break
				}
			}
		}
		if next < 0 {
			break
		}
		path = append(path, next)
	}
	res.Path = path
	res.Ordered = true
	for k, idx := range path {
		if source[idx] == "input" && idx != k {
			res.Ordered = false
		}
	}
	verify := func(certs []*x509.Certificate) error {
		inter := x509.NewCertPool()
		for i, c := range certs {
			if i != leaf {
				inter.AddCert(c)
			}
		}
		_, err := all[leaf].Verify(x509.VerifyOptions{Intermediates: inter, Roots: tlsx.RootsForTests, CurrentTime: now, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
		return err //nolint:wrapcheck // summarized below
	}
	if err := verify(all); err != nil {
		res.VerifyError = tlsx.Verify(all, "", now).Error()
	} else {
		res.Trusted = true
		res.Complete = verify(input) == nil
	}
	if host != "" {
		ok := all[leaf].VerifyHostname(host) == nil
		res.HostOK = &ok
	}
	for i, c := range all {
		cc := ChainCert{Cert: tlsx.Summarize(c, now), Source: source[i], InputIndex: -1, Expired: now.After(c.NotAfter) || now.Before(c.NotBefore), Role: "extra"}
		if source[i] == "input" {
			cc.InputIndex = i
		}
		if len(c.IssuingCertificateURL) > 0 {
			cc.AIA = c.IssuingCertificateURL[0]
		}
		if k := slices.Index(path, i); k >= 0 {
			switch {
			case k == 0:
				cc.Role = "leaf"
			case selfSigned(c):
				cc.Role = "root"
			default:
				cc.Role = "intermediate"
			}
			if k+1 < len(path) {
				ok := issuedBy(c, all[path[k+1]])
				cc.SignatureOK = &ok
			}
		}
		res.Certs = append(res.Certs, cc)
	}
	if n := len(all) - len(input); n > 0 && res.Trusted && !res.Complete {
		res.Notes = append(res.Notes, "missing-intermediates")
	}
	return res
}

func (t *ChainTool) handle(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PEM  string `json:"pem"`
		Host string `json:"host"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxChainPEMLen)).Decode(&in); err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("pem", `expected JSON {"pem": "-----BEGIN CERTIFICATE-----…", "host": "optional"}`))
		return
	}
	certs, err := ParsePEMCerts(in.PEM)
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("pem", err.Error()))
		return
	}
	if len(certs) == 0 || len(certs) > MaxChainCerts {
		httpapi.Error(w, r, httpapi.InvalidInput("pem", "paste between 1 and 12 PEM certificates (-----BEGIN CERTIFICATE-----)"))
		return
	}
	res := t.Analyze(r.Context(), certs, strings.TrimSpace(strings.ToLower(in.Host)))
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
