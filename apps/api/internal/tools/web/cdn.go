package web

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// CDNTool is WEB-16: which CDN, WAF / bot protection and hosting platform serve a site.
type CDNTool struct {
	deps tools.Deps
	// Resolver overrides the system resolver for the CNAME chain (tests).
	Resolver string
}

func (*CDNTool) ID() string                                { return "web.cdn" }
func (*CDNTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *CDNTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(30*time.Second)).Get("/http/cdn", t.handle)
}

// Detection is one recognised service with the evidence for it.
type Detection struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"` // cdn, waf, bot, hosting, cache, server
	Evidence []string `json:"evidence"`
}

// CDNResult is the WEB-16 payload.
type CDNResult struct {
	URL         string      `json:"url"`
	Status      int         `json:"status"`
	RemoteAddr  string      `json:"remoteAddr"`
	CNAMEs      []string    `json:"cnames"`
	Server      string      `json:"server"`
	CacheStatus string      `json:"cacheStatus"`
	Issuer      string      `json:"issuer"`
	Detections  []Detection `json:"detections"`
}

type signature struct {
	name, kind string
	headers    []string // header name (lowercase), or "name=substring" on the value
	cookies    []string // cookie name prefixes
	cnames     []string // CNAME suffixes
	issuers    []string // certificate issuer substrings
}

var signatures = []signature{
	{name: "Cloudflare", kind: "cdn", headers: []string{"cf-ray", "server=cloudflare", "cf-cache-status"}, cookies: []string{"__cf_bm", "cf_clearance", "__cflb"}, cnames: []string{".cdn.cloudflare.net"}},
	{name: "Amazon CloudFront", kind: "cdn", headers: []string{"x-amz-cf-id", "x-amz-cf-pop", "via=cloudfront"}, cnames: []string{".cloudfront.net"}},
	{name: "Fastly", kind: "cdn", headers: []string{"x-fastly-request-id", "fastly-debug-digest", "x-served-by=cache-"}, cnames: []string{".fastly.net", ".fastlylb.net"}},
	{name: "Akamai", kind: "cdn", headers: []string{"x-akamai-transformed", "akamai-grn", "server=akamaighost", "x-akamai-request-id"}, cnames: []string{".akamaiedge.net", ".edgekey.net", ".edgesuite.net", ".akamaized.net", ".akamai.net"}},
	{name: "Azure Front Door / CDN", kind: "cdn", headers: []string{"x-azure-ref", "x-fd-healthprobe", "x-msedge-ref"}, cnames: []string{".azurefd.net", ".azureedge.net", ".trafficmanager.net", ".t-msedge.net"}},
	{name: "Google Cloud CDN / Front End", kind: "cdn", headers: []string{"via=google", "server=gws", "server=esf", "x-goog-"}, cnames: []string{"ghs.googlehosted.com", ".googleusercontent.com"}},
	{name: "Bunny CDN", kind: "cdn", headers: []string{"server=bunnycdn", "cdn-pullzone", "cdn-requestid"}, cnames: []string{".b-cdn.net"}},
	{name: "CDN77", kind: "cdn", headers: []string{"x-77-nzt", "x-77-cache"}, cnames: []string{".cdn77.org", ".cdn77.net"}},
	{name: "Edgio / Edgecast", kind: "cdn", headers: []string{"server=ecacc", "server=ecs", "x-ec-custom-error"}, cnames: []string{".edgecastcdn.net", ".systemcdn.net"}},
	{name: "KeyCDN", kind: "cdn", headers: []string{"server=keycdn"}, cnames: []string{".kxcdn.com"}},
	{name: "Gcore", kind: "cdn", headers: []string{"x-id-fe", "server=gcore"}, cnames: []string{".gcdn.co"}},
	{name: "Imperva / Incapsula", kind: "waf", headers: []string{"x-iinfo", "x-cdn=imperva", "x-cdn=incapsula"}, cookies: []string{"incap_ses_", "visid_incap_", "nlbi_"}, cnames: []string{".incapdns.net", ".impervadns.net"}},
	{name: "Sucuri", kind: "waf", headers: []string{"x-sucuri-id", "x-sucuri-cache", "server=sucuri"}, cnames: []string{".sucuri.net"}},
	{name: "F5 BIG-IP", kind: "waf", headers: []string{"server=big-ip", "x-wa-info"}, cookies: []string{"bigipserver", "ts01", "f5_cspm"}},
	{name: "Barracuda WAF", kind: "waf", cookies: []string{"barra_counter_session", "bni__barracuda_lb_cookie"}},
	{name: "AWS WAF", kind: "waf", headers: []string{"x-amzn-waf-action"}, cookies: []string{"aws-waf-token"}},
	{name: "Akamai Bot Manager", kind: "bot", cookies: []string{"ak_bmsc", "bm_sv", "_abck", "bm_sz"}},
	{name: "DataDome", kind: "bot", headers: []string{"x-datadome", "x-dd-b"}, cookies: []string{"datadome"}},
	{name: "HUMAN (PerimeterX)", kind: "bot", cookies: []string{"_px", "_pxvid", "_pxhd"}},
	{name: "Vercel", kind: "hosting", headers: []string{"x-vercel-id", "server=vercel", "x-vercel-cache"}, cnames: []string{".vercel-dns.com", ".vercel.app"}},
	{name: "Netlify", kind: "hosting", headers: []string{"x-nf-request-id", "server=netlify"}, cnames: []string{".netlify.app", ".netlify.com"}},
	{name: "GitHub Pages", kind: "hosting", headers: []string{"x-github-request-id", "server=github.com"}, cnames: []string{".github.io"}},
	{name: "Shopify", kind: "hosting", headers: []string{"x-shopid", "x-shopify-stage"}, cnames: []string{"shops.myshopify.com"}},
	{name: "WP Engine", kind: "hosting", headers: []string{"x-powered-by=wp engine", "wpe-backend"}, cnames: []string{".wpengine.com", ".wpenginepowered.com"}},
	{name: "Heroku", kind: "hosting", headers: []string{"via=vegur"}, cnames: []string{".herokudns.com", ".herokuapp.com"}},
	{name: "Varnish", kind: "cache", headers: []string{"x-varnish", "via=varnish"}},
	{name: "Let's Encrypt certificate", kind: "tls", issuers: []string{"Let's Encrypt"}},
}

// Detect matches headers, cookie names, CNAMEs and the certificate issuer against the signatures.
func Detect(hdrs []httpcore.Header, cnames []string, issuer string) []Detection {
	var cookies []string
	for _, h := range hdrs {
		if equalFold(h.Name, "Set-Cookie") {
			name, _, _ := strings.Cut(h.Value, "=")
			cookies = append(cookies, strings.ToLower(strings.TrimSpace(name)))
		}
	}
	out := []Detection{}
	for _, s := range signatures {
		var ev []string
		for _, sig := range s.headers {
			name, sub, hasSub := strings.Cut(sig, "=")
			for _, h := range hdrs {
				ln := strings.ToLower(h.Name)
				if (hasSub && ln == name && strings.Contains(strings.ToLower(h.Value), sub)) ||
					(!hasSub && (ln == name || (strings.HasSuffix(name, "-") && strings.HasPrefix(ln, name)))) {
					ev = append(ev, h.Name+": "+truncateValue(h.Value))
					break
				}
			}
		}
		for _, c := range s.cookies {
			for _, have := range cookies {
				if strings.HasPrefix(have, c) {
					ev = append(ev, "cookie "+have)
					break
				}
			}
		}
		for _, suffix := range s.cnames {
			for _, cn := range cnames {
				if strings.HasSuffix(cn, suffix) || cn == strings.TrimPrefix(suffix, ".") {
					ev = append(ev, "CNAME "+cn)
				}
			}
		}
		for _, is := range s.issuers {
			if strings.Contains(issuer, is) {
				ev = append(ev, "issuer "+issuer)
			}
		}
		if len(ev) > 0 {
			out = append(out, Detection{Name: s.name, Kind: s.kind, Evidence: slices.Compact(ev)})
		}
	}
	return out
}

func truncateValue(v string) string {
	if len(v) > 80 {
		return v[:80] + "…"
	}
	return v
}

// cacheStatusHeaders, most specific first.
var cacheStatusHeaders = []string{"Cache-Status", "CF-Cache-Status", "X-Cache", "X-Cache-Status", "X-Vercel-Cache", "X-Proxy-Cache", "CDN-Cache", "X-Drupal-Cache", "X-Varnish-Cache"}

func cacheStatus(h httpcore.Hop) string {
	for _, n := range cacheStatusHeaders {
		if v, ok := header(h, n); ok {
			return n + ": " + v
		}
	}
	return ""
}

// cnameChain follows CNAMEs of host (at most 8).
func cnameChain(ctx context.Context, srv dnsx.Server, host string) []string {
	out := []string{}
	cur := host
	for range 8 {
		msg, _, err := dnsx.Query(ctx, srv, cur, dns.TypeCNAME)
		if err != nil {
			break
		}
		next := ""
		for _, rr := range msg.Answer {
			if c, ok := rr.(*dns.CNAME); ok {
				next = strings.ToLower(strings.TrimSuffix(c.Target, "."))
			}
		}
		if next == "" || slices.Contains(out, next) {
			break
		}
		out = append(out, next)
		cur = next
	}
	return out
}

func (t *CDNTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: raw, Method: "GET", FollowRedirects: true})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	last := hops[len(hops)-1]
	res := CDNResult{URL: last.URL, Status: last.Status, RemoteAddr: last.RemoteAddr, CNAMEs: []string{}}
	res.Server, _ = header(last, "Server")
	res.CacheStatus = cacheStatus(last)
	if last.TLS != nil {
		res.Issuer = last.TLS.Issuer
	}
	if u, err := url.Parse(last.URL); err == nil {
		if srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, cmp.Or(t.Resolver, "system"), dnsx.UDP, 0); err == nil {
			res.CNAMEs = cnameChain(r.Context(), srv, u.Hostname())
		}
	}
	// Headers of every hop count: a WAF may only show on the first response.
	var all []httpcore.Header
	for _, h := range hops {
		all = append(all, h.Headers...)
	}
	res.Detections = Detect(all, res.CNAMEs, res.Issuer)
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}
