package web

import (
	"bufio"
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/netx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// WellKnownTool is WEB-09: robots.txt, sitemap and security.txt of a site.
type WellKnownTool struct {
	deps tools.Deps
	Now  func() time.Time
}

func (*WellKnownTool) ID() string                                { return "web.wellknown" }
func (*WellKnownTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *WellKnownTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Now == nil {
		t.Now = time.Now
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(60*time.Second)).Get("/http/wellknown", t.handle)
}

// File is one fetched well-known file.
type File struct {
	URL         string `json:"url"`
	Found       bool   `json:"found"`
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Bytes       int    `json:"bytes"`
	Error       string `json:"error"`
}

// RobotsGroup is one User-agent group.
type RobotsGroup struct {
	Agents      []string `json:"agents"`
	Allow       int      `json:"allow"`
	Disallow    int      `json:"disallow"`
	DisallowAll bool     `json:"disallowAll"`
	CrawlDelay  string   `json:"crawlDelay"`
}

// Robots is the parsed robots.txt.
type Robots struct {
	File      File          `json:"file"`
	Groups    []RobotsGroup `json:"groups"`
	Sitemaps  []string      `json:"sitemaps"`
	BlocksAll bool          `json:"blocksAll"`
}

// SecurityTxt is the parsed RFC 9116 file.
type SecurityTxt struct {
	File     File                `json:"file"`
	Fields   map[string][]string `json:"fields"`
	Signed   bool                `json:"signed"`
	Problems []string            `json:"problems"`
	Warnings []string            `json:"warnings"`
}

// Sitemap summarizes the first sitemap found.
type Sitemap struct {
	File    File   `json:"file"`
	Kind    string `json:"kind"`
	Entries int    `json:"entries"`
	Newest  string `json:"newest"`
	Capped  bool   `json:"capped"`
}

// WellKnownResult is the WEB-09 payload.
type WellKnownResult struct {
	Origin   string      `json:"origin"`
	Robots   Robots      `json:"robots"`
	Security SecurityTxt `json:"security"`
	Sitemap  Sitemap     `json:"sitemap"`
}

// get fetches one file; err is set only when the request could not be made at all (policy,
// resolution), which for the first file means the whole site is unusable.
func (t *WellKnownTool) get(r *http.Request, u string) (File, string, error) {
	f := File{URL: u}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: u, FollowRedirects: true, PreviewBytes: 512 << 10})
	if err != nil {
		f.Error = httpapi.ToAPIError(err).Message
		return f, "", err
	}
	last := hops[len(hops)-1]
	f.URL, f.Status, f.ContentType, f.Bytes, f.Error = last.URL, last.Status, last.ContentType, last.BodyBytes, last.Error
	f.Found = last.Status == http.StatusOK
	if !f.Found {
		return f, "", nil
	}
	return f, last.BodyPreview, nil
}

// ParseRobots reads robots.txt groups (RFC 9309).
func ParseRobots(body string) ([]RobotsGroup, []string) {
	groups := []RobotsGroup{}
	sitemaps := []string{}
	var cur *RobotsGroup
	lastWasAgent := false
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := strings.TrimSpace(strings.SplitN(sc.Text(), "#", 2)[0])
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, val = strings.ToLower(strings.TrimSpace(key)), strings.TrimSpace(val)
		switch key {
		case "user-agent":
			if cur == nil || !lastWasAgent {
				groups = append(groups, RobotsGroup{Agents: []string{}})
				cur = &groups[len(groups)-1]
			}
			cur.Agents = append(cur.Agents, val)
			lastWasAgent = true
			continue
		case "sitemap":
			if len(sitemaps) < 50 {
				sitemaps = append(sitemaps, val)
			}
		case "allow":
			if cur != nil && val != "" {
				cur.Allow++
			}
		case "disallow":
			if cur != nil && val != "" {
				cur.Disallow++
				cur.DisallowAll = cur.DisallowAll || val == "/"
			}
		case "crawl-delay":
			if cur != nil {
				cur.CrawlDelay = val
			}
		}
		lastWasAgent = false
	}
	return groups, sitemaps
}

// ParseSecurityTxt checks RFC 9116 fields.
func ParseSecurityTxt(body, servedAt string, now time.Time) SecurityTxt {
	s := SecurityTxt{Fields: map[string][]string{}, Problems: []string{}, Warnings: []string{}}
	s.Signed = strings.Contains(body, "-----BEGIN PGP SIGNED MESSAGE-----")
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-----") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok || strings.Contains(key, " ") {
			continue
		}
		k := http.CanonicalHeaderKey(strings.TrimSpace(key))
		if len(s.Fields[k]) < 20 {
			s.Fields[k] = append(s.Fields[k], strings.TrimSpace(val))
		}
	}
	if len(s.Fields["Contact"]) == 0 {
		s.Problems = append(s.Problems, "no-contact")
	}
	switch exp := s.Fields["Expires"]; {
	case len(exp) == 0:
		s.Problems = append(s.Problems, "no-expires")
	case len(exp) > 1:
		s.Problems = append(s.Problems, "multiple-expires")
	default:
		ts, err := time.Parse(time.RFC3339, exp[0])
		switch {
		case err != nil:
			s.Problems = append(s.Problems, "bad-expires")
		case ts.Before(now):
			s.Problems = append(s.Problems, "expired")
		case ts.Sub(now) > 366*24*time.Hour:
			s.Warnings = append(s.Warnings, "expires-far")
		}
	}
	if !strings.Contains(servedAt, "/.well-known/security.txt") {
		s.Warnings = append(s.Warnings, "legacy-location")
	}
	if !s.Signed {
		s.Warnings = append(s.Warnings, "unsigned")
	}
	return s
}

// SummarizeSitemap counts <url> or <sitemap> entries (capped) and finds the newest lastmod.
func SummarizeSitemap(body string) (kind string, entries int, newest string, capped bool) {
	// Token streaming over a body capped at 512 KiB; encoding/xml never expands external entities.
	dec := xml.NewDecoder(strings.NewReader(body)) //nolint:gosec // see above
	dec.Strict = false
	inLastmod := false
	for {
		tok, err := dec.Token()
		if err != nil { // io.EOF included
			break
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch el.Name.Local {
			case "urlset", "sitemapindex":
				kind = el.Name.Local
			case "url", "sitemap":
				entries++
			case "lastmod":
				inLastmod = true
			}
		case xml.CharData:
			if inLastmod {
				if v := strings.TrimSpace(string(el)); v > newest {
					newest = v
				}
			}
		case xml.EndElement:
			inLastmod = false
		}
	}
	capped = len(body) >= 512<<10
	return kind, entries, newest, capped
}

func (t *WellKnownTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	u, err := netx.ParseHTTPURL(raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	origin := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	out := WellKnownResult{Origin: origin}

	f, body, err := t.get(r, origin+"/robots.txt")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out.Robots = Robots{File: f, Groups: []RobotsGroup{}, Sitemaps: []string{}}
	if f.Found {
		out.Robots.Groups, out.Robots.Sitemaps = ParseRobots(body)
		for _, g := range out.Robots.Groups {
			for _, a := range g.Agents {
				out.Robots.BlocksAll = out.Robots.BlocksAll || (a == "*" && g.DisallowAll)
			}
		}
	}

	f, body, _ = t.get(r, origin+"/.well-known/security.txt")
	if !f.Found {
		if alt, altBody, _ := t.get(r, origin+"/security.txt"); alt.Found {
			f, body = alt, altBody
		}
	}
	out.Security = SecurityTxt{File: f, Fields: map[string][]string{}, Problems: []string{}, Warnings: []string{}}
	if f.Found {
		s := ParseSecurityTxt(body, f.URL, t.Now())
		s.File = f
		out.Security = s
	}

	sm := origin + "/sitemap.xml"
	if len(out.Robots.Sitemaps) > 0 {
		if su, err := u.Parse(out.Robots.Sitemaps[0]); err == nil {
			sm = su.String() // fetched through the same policy-checked client
		}
	}
	f, body, _ = t.get(r, sm)
	out.Sitemap = Sitemap{File: f}
	if f.Found {
		out.Sitemap.Kind, out.Sitemap.Entries, out.Sitemap.Newest, out.Sitemap.Capped = SummarizeSitemap(body)
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}
