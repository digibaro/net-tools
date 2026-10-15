package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

const robotsTxt = `# robots
User-agent: *
Disallow: /admin
Disallow: /tmp
Allow: /tmp/public

User-agent: BadBot
User-agent: OtherBot
Disallow: /
Crawl-delay: 10

Sitemap: /maps/sitemap.xml
`

const securityTxt = `Contact: mailto:security@example.com
Contact: https://example.com/report
Expires: 2027-01-01T00:00:00Z
Preferred-Languages: en, nl
Policy: https://example.com/policy
`

const sitemapXML = `<?xml version="1.0"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"><url><loc>https://example.com/</loc><lastmod>2026-09-01</lastmod></url><url><loc>https://example.com/a</loc><lastmod>2026-10-01</lastmod></url></urlset>`

func TestWellKnown(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	groups, maps := ParseRobots(robotsTxt)
	if len(groups) != 2 || groups[0].Disallow != 2 || groups[0].Allow != 1 || !groups[1].DisallowAll || len(groups[1].Agents) != 2 || groups[1].CrawlDelay != "10" || maps[0] != "/maps/sitemap.xml" {
		t.Errorf("robots = %+v %v", groups, maps)
	}
	s := ParseSecurityTxt(securityTxt, "https://example.com/.well-known/security.txt", now)
	if len(s.Problems) != 0 || len(s.Fields["Contact"]) != 2 || !slices.Contains(s.Warnings, "unsigned") {
		t.Errorf("security = %+v", s)
	}
	if s := ParseSecurityTxt("Contact: x\nExpires: 2020-01-01T00:00:00Z\n", "https://e/security.txt", now); !slices.Contains(s.Problems, "expired") || !slices.Contains(s.Warnings, "legacy-location") {
		t.Errorf("expired = %+v", s)
	}
	if s := ParseSecurityTxt("Policy: x\n", "https://e/.well-known/security.txt", now); !slices.Contains(s.Problems, "no-contact") || !slices.Contains(s.Problems, "no-expires") {
		t.Errorf("empty = %+v", s)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte(robotsTxt))
		case "/.well-known/security.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(securityTxt))
		case "/maps/sitemap.xml":
			_, _ = w.Write([]byte(sitemapXML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	var out WellKnownResult
	tooltest.New(t, &WellKnownTool{Now: func() time.Time { return now }}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/http/wellknown?url="+url.QueryEscape(srv.URL+"/some/page")).Must(t).Decode(t, &out)
	if !out.Robots.File.Found || out.Robots.BlocksAll || !out.Security.File.Found || out.Sitemap.Kind != "urlset" || out.Sitemap.Entries != 2 || out.Sitemap.Newest != "2026-10-01" {
		t.Errorf("out = %+v", out)
	}
	tooltest.New(t, &WellKnownTool{}, tooltest.Options{}).Get("/api/v1/http/wellknown?url="+url.QueryEscape("http://10.0.0.1/")).ExpectError(t, "TARGET_PRIVATE_DENIED")
}
