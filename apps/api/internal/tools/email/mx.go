package email

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tlsx"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/rttstats"
)

// MaxMX bounds probed mail servers.
const MaxMX = 10

// MXTool is EM-01.
type MXTool struct {
	deps tools.Deps
	res  resolver
	// Port overrides 25 (tests).
	Port int
}

func (*MXTool) ID() string                                { return "email.mx" }
func (*MXTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *MXTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.res == nil {
		t.res = sysResolver{}
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(60*time.Second)).Get("/email/mx", t.handle)
}

// SMTPProbe is the result of talking to one mail server.
type SMTPProbe struct {
	Address    string        `json:"address"`
	Connected  bool          `json:"connected"`
	ConnectMs  float64       `json:"connectMs"`
	Banner     string        `json:"banner"`
	Extensions []string      `json:"extensions"`
	StartTLS   bool          `json:"starttls"`
	TLS        *tlsx.Session `json:"tls"`
	Error      string        `json:"error"`
}

// MXHost is one MX record with its probes.
type MXHost struct {
	Preference uint16      `json:"preference"`
	Host       string      `json:"host"`
	Addresses  []string    `json:"addresses"`
	Probes     []SMTPProbe `json:"probes"`
	Error      string      `json:"error"`
}

// MXResult is the response payload.
type MXResult struct {
	Domain   string   `json:"domain"`
	Hosts    []MXHost `json:"hosts"`
	NullMX   bool     `json:"nullMx"`
	Implicit bool     `json:"implicit"`
	Probed   bool     `json:"probed"`
}

func (t *MXTool) handle(w http.ResponseWriter, r *http.Request) {
	domain, ok := domainParam(w, r, "domain")
	if !ok {
		return
	}
	probe := r.URL.Query().Get("probe") != "false"
	ctx := r.Context()
	mxs, rcode, err := t.res.MX(ctx, domain)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	res := MXResult{Domain: domain, Hosts: []MXHost{}, Probed: probe}
	if len(mxs) == 1 && (mxs[0].Mx == "." || mxs[0].Mx == "") {
		res.NullMX = true // RFC 7505: domain accepts no mail
		httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
		return
	}
	if len(mxs) == 0 {
		if rcode == "NXDOMAIN" {
			httpapi.Error(w, r, httpapi.InvalidInput("domain", "the domain does not exist"))
			return
		}
		res.Implicit = true // RFC 5321 §5.1: fall back to the domain's A/AAAA
		mxs = append(mxs, &dns.MX{Preference: 0, Mx: domain + "."})
	}
	for i, mx := range mxs {
		if i >= MaxMX {
			break
		}
		res.Hosts = append(res.Hosts, MXHost{Preference: mx.Preference, Host: strings.TrimSuffix(strings.ToLower(mx.Mx), "."), Addresses: []string{}, Probes: []SMTPProbe{}})
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i := range res.Hosts {
		h := &res.Hosts[i]
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			t.probeHost(ctx, h, probe)
		})
	}
	wg.Wait()
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: domain}})
}

func (t *MXTool) probeHost(ctx context.Context, h *MXHost, probe bool) {
	tgt, err := t.deps.Target.Resolve(ctx, h.Host, target.Opts{Field: "domain"})
	if err != nil {
		h.Error = httpapi.ToAPIError(err).Message
		return
	}
	for _, a := range tgt.Resolved {
		h.Addresses = append(h.Addresses, a.String())
	}
	if !probe {
		return
	}
	port := t.Port
	if port == 0 {
		port = 25
	}
	// Probe the first IPv4 and first IPv6 address.
	var picked []netip.Addr
	var has4, has6 bool
	for _, a := range tgt.Resolved {
		if a.Is4() && !has4 {
			picked, has4 = append(picked, a), true
		} else if a.Is6() && !has6 {
			picked, has6 = append(picked, a), true
		}
	}
	for _, a := range picked {
		h.Probes = append(h.Probes, ProbeSMTP(ctx, netip.AddrPortFrom(a, uint16(port)).String(), h.Host)) //nolint:gosec // port is 25 or a test port
	}
}

// ProbeSMTP reads the banner, sends EHLO and, if offered, upgrades with STARTTLS.
func ProbeSMTP(ctx context.Context, addr, mxName string) SMTPProbe {
	p := SMTPProbe{Address: addr, Extensions: []string{}}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	d := net.Dialer{Timeout: 8 * time.Second}
	start := time.Now()
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		p.Error = "cannot connect to port 25 (often blocked by the server's network provider)"
		return p
	}
	defer c.Close()
	p.Connected, p.ConnectMs = true, rttstats.Ms(time.Since(start))
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetDeadline(dl)
	}
	r := bufio.NewReader(c)
	reply := func(code string) ([]string, error) {
		var lines []string
		for {
			l, err := r.ReadString('\n')
			if err != nil {
				return lines, fmt.Errorf("read reply: %w", err)
			}
			l = strings.TrimRight(l, "\r\n")
			if !strings.HasPrefix(l, code) {
				return lines, fmt.Errorf("unexpected reply %q", l)
			}
			lines = append(lines, l[min(4, len(l)):])
			if len(l) < 4 || l[3] != '-' {
				return lines, nil
			}
		}
	}
	banner, err := reply("220")
	if err != nil {
		p.Error = "no SMTP banner"
		return p
	}
	p.Banner = strings.Join(banner, " ")
	_, _ = fmt.Fprint(c, "EHLO nettools.invalid\r\n")
	ext, err := reply("250")
	if err != nil {
		p.Error = "EHLO rejected"
		return p
	}
	if len(ext) > 1 {
		p.Extensions = ext[1:]
	}
	for _, e := range p.Extensions {
		if strings.EqualFold(strings.Fields(e + " ")[0], "STARTTLS") {
			p.StartTLS = true
		}
	}
	if !p.StartTLS {
		_, _ = fmt.Fprint(c, "QUIT\r\n")
		return p
	}
	_, _ = fmt.Fprint(c, "STARTTLS\r\n")
	if _, err := reply("220"); err != nil {
		p.Error = "STARTTLS refused"
		return p
	}
	tc := tls.Client(c, &tls.Config{ServerName: mxName, InsecureSkipVerify: true, MinVersion: tls.VersionTLS10}) //nolint:gosec // verified below
	hstart := time.Now()
	if err := tc.HandshakeContext(ctx); err != nil {
		p.Error = "TLS handshake failed after STARTTLS"
		return p
	}
	st := tc.ConnectionState()
	s := &tlsx.Session{Version: tls.VersionName(st.Version), Cipher: tls.CipherSuiteName(st.CipherSuite), ServerName: mxName, StartTLS: "smtp", ConnectedTo: addr, HandshakeMs: rttstats.Ms(time.Since(hstart)), Chain: []tlsx.Cert{}}
	for _, cert := range st.PeerCertificates {
		s.Chain = append(s.Chain, tlsx.Summarize(cert, time.Now()))
	}
	if err := tlsx.Verify(st.PeerCertificates, mxName, time.Now()); err != nil {
		s.VerifyError = err.Error()
	} else {
		s.Verified = true
	}
	p.TLS = s
	_, _ = fmt.Fprint(tc, "QUIT\r\n")
	return p
}
