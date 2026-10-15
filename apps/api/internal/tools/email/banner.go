package email

import (
	"cmp"
	"context"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// BannerTool is EM-10: SMTP banner, EHLO name and forward-confirmed reverse DNS of a mail server.
type BannerTool struct {
	deps tools.Deps
	// Port overrides 25 (tests); Resolver overrides the system resolver (tests).
	Port     int
	Resolver string
}

func (*BannerTool) ID() string                                { return "email.banner" }
func (*BannerTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *BannerTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Get("/email/smtp-banner", t.handle)
}

// BannerCheck is one pass/fail line.
type BannerCheck struct {
	ID     string `json:"id"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// BannerResult is the EM-10 payload.
type BannerResult struct {
	Host        string        `json:"host"`
	Address     string        `json:"address"`
	PTR         []string      `json:"ptr"`
	Forward     []string      `json:"forward"`
	FCrDNS      bool          `json:"fcrdns"`
	BannerHost  string        `json:"bannerHost"`
	EHLOHost    string        `json:"ehloHost"`
	Probe       SMTPProbe     `json:"probe"`
	Checks      []BannerCheck `json:"checks"`
	Passed      int           `json:"passed"`
	SMTPBlocked bool          `json:"smtpBlocked"`
}

// isFQDN: at least two labels and not an address literal.
func isFQDN(s string) bool {
	if _, err := netip.ParseAddr(strings.Trim(s, "[]")); err == nil {
		return false
	}
	_, ok := dns.IsDomainName(s)
	return ok && strings.Count(strings.Trim(s, "."), ".") >= 1
}

func firstWord(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return strings.TrimSuffix(strings.ToLower(f[0]), ".")
	}
	return ""
}

func (t *BannerTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("host", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	tgt, err := t.deps.Target.Resolve(ctx, strings.TrimSpace(raw), target.Opts{Field: "host"})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	addr := tgt.Resolved[0]
	for _, a := range tgt.Resolved {
		if a.Is4() {
			addr = a
			break
		}
	}
	srv, err := dnsx.ResolveServer(ctx, t.deps.Target, cmp.Or(t.Resolver, "system"), dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	res := BannerResult{Host: tgt.Host, Address: addr.String(), PTR: []string{}, Forward: []string{}, Checks: []BannerCheck{}}
	t.reverse(ctx, srv, addr, &res)
	port := t.Port
	if port == 0 {
		port = 25
	}
	name := tgt.Host
	if tgt.IsIP && len(res.PTR) > 0 {
		name = res.PTR[0]
	}
	res.Probe = ProbeSMTP(ctx, netip.AddrPortFrom(addr, uint16(port)).String(), name) //nolint:gosec // 25 or a test port
	res.SMTPBlocked = !res.Probe.Connected
	res.BannerHost, res.EHLOHost = firstWord(res.Probe.Banner), firstWord(res.Probe.Greeting)
	res.Checks = bannerChecks(res)
	for _, c := range res.Checks {
		if c.Pass {
			res.Passed++
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw, Used: addr.String()}})
}

func (t *BannerTool) reverse(ctx context.Context, srv dnsx.Server, addr netip.Addr, res *BannerResult) {
	arpa, _ := dns.ReverseAddr(addr.String())
	msg, _, err := dnsx.Query(ctx, srv, arpa, dns.TypePTR)
	if err != nil {
		return
	}
	for _, rr := range msg.Answer {
		if p, ok := rr.(*dns.PTR); ok {
			res.PTR = append(res.PTR, strings.TrimSuffix(strings.ToLower(p.Ptr), "."))
		}
	}
	for _, n := range res.PTR {
		fwd, _ := dnsx.Addresses(ctx, srv, n)
		for _, a := range fwd {
			res.Forward = append(res.Forward, a.String())
		}
		if slices.Contains(fwd, addr) {
			res.FCrDNS = true
		}
	}
}

func bannerChecks(r BannerResult) []BannerCheck {
	c := func(id string, pass bool, detail string) BannerCheck {
		return BannerCheck{ID: id, Pass: pass, Detail: detail}
	}
	ptr := strings.Join(r.PTR, ", ")
	out := []BannerCheck{
		c("ptr", len(r.PTR) > 0, ptr),
		c("fcrdns", r.FCrDNS, strings.Join(r.Forward, ", ")),
	}
	if !r.Probe.Connected {
		return out
	}
	out = append(out,
		c("banner", r.Probe.Banner != "", r.Probe.Banner),
		c("bannerFqdn", isFQDN(r.BannerHost), r.BannerHost),
		c("bannerPtr", r.BannerHost != "" && slices.Contains(r.PTR, r.BannerHost), r.BannerHost),
		c("ehloPtr", r.EHLOHost != "" && slices.Contains(r.PTR, r.EHLOHost), r.EHLOHost),
		c("starttls", r.Probe.StartTLS, ""),
	)
	if r.Probe.TLS != nil {
		out = append(out, c("certificate", r.Probe.TLS.Verified, r.Probe.TLS.VerifyError))
	}
	return out
}
