package web

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/net/html"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// MixedTool is WEB-18: http:// subresources and forms on an https page.
type MixedTool struct{ deps tools.Deps }

func (*MixedTool) ID() string                                { return "web.mixed" }
func (*MixedTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *MixedTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(30*time.Second)).Get("/http/mixed-content", t.handle)
}

// MaxMixedItems bounds the reported references.
const MaxMixedItems = 500

// MixedItem is one insecure reference. Kind: active (blocked by browsers), passive (images and
// media: upgraded or blocked), form (submits in clear text), link (navigation only, informative).
type MixedItem struct {
	Tag  string `json:"tag"`
	Attr string `json:"attr"`
	URL  string `json:"url"`
	Kind string `json:"kind"`
}

// MixedResult is the WEB-18 payload.
type MixedResult struct {
	URL                     string      `json:"url"`
	Status                  int         `json:"status"`
	ContentType             string      `json:"contentType"`
	BodyBytes               int         `json:"bodyBytes"`
	Truncated               bool        `json:"truncated"`
	UpgradeInsecureRequests bool        `json:"upgradeInsecureRequests"`
	Items                   []MixedItem `json:"items"`
	Active                  int         `json:"active"`
	Passive                 int         `json:"passive"`
	Forms                   int         `json:"forms"`
	Links                   int         `json:"links"`
	Resources               int         `json:"resources"`
}

var cssURL = regexp.MustCompile(`(?i)url\(\s*['"]?(http://[^'")\s]+)`)

// kindOf classifies a reference by tag and attribute.
func kindOf(tag string, n *html.Node) string {
	switch tag {
	case "script", "iframe", "frame", "object", "embed", "applet":
		return "active"
	case "link":
		rel := strings.ToLower(attrOf(n, "rel"))
		if strings.Contains(rel, "stylesheet") || strings.Contains(rel, "preload") || strings.Contains(rel, "modulepreload") || strings.Contains(rel, "manifest") {
			return "active"
		}
		if strings.Contains(rel, "icon") {
			return "passive"
		}
		return "link"
	case "img", "audio", "video", "source", "track", "picture", "input":
		return "passive"
	case "form", "button":
		return "form"
	case "a", "area":
		return "link"
	}
	return "passive"
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

var refAttrs = map[string]bool{"src": true, "href": true, "data": true, "action": true, "formaction": true, "poster": true, "srcset": true}

// FindMixed walks an HTML document (base = page URL) and returns every http:// reference.
func FindMixed(doc *html.Node, base *url.URL) (items []MixedItem, resources int, upgrade bool) {
	var walk func(n *html.Node)
	add := func(tag, attr, ref string, n *html.Node) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		resources++
		u, err := base.Parse(ref)
		if err != nil || u.Scheme != "http" || len(items) >= MaxMixedItems {
			return
		}
		items = append(items, MixedItem{Tag: tag, Attr: attr, URL: u.String(), Kind: kindOf(tag, n)})
	}
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := strings.ToLower(n.Data)
			if tag == "base" {
				if b, err := base.Parse(attrOf(n, "href")); err == nil && attrOf(n, "href") != "" {
					base = b
				}
			}
			if tag == "meta" && strings.EqualFold(attrOf(n, "http-equiv"), "content-security-policy") && strings.Contains(strings.ToLower(attrOf(n, "content")), "upgrade-insecure-requests") {
				upgrade = true
			}
			canonical := tag == "link" && strings.Contains(strings.ToLower(attrOf(n, "rel")), "canonical")
			for _, a := range n.Attr {
				k := strings.ToLower(a.Key)
				switch {
				case k == "srcset":
					for _, c := range strings.Split(a.Val, ",") {
						if f := strings.Fields(c); len(f) > 0 {
							add(tag, k, f[0], n)
						}
					}
				case refAttrs[k] && tag != "base" && !canonical:
					add(tag, k, a.Val, n)
				case k == "style":
					for _, m := range cssURL.FindAllStringSubmatch(a.Val, -1) {
						add(tag, "style", m[1], n)
					}
				}
			}
			if tag == "style" && n.FirstChild != nil {
				for _, m := range cssURL.FindAllStringSubmatch(n.FirstChild.Data, -1) {
					add("style", "url()", m[1], n)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return items, resources, upgrade
}

func (t *MixedTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: raw, Method: "GET", FollowRedirects: true, PreviewBytes: int(t.deps.Config.Limits.HTTPMaxBodyBytes)})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	last := hops[len(hops)-1]
	page, err := url.Parse(last.URL)
	if err != nil || page.Scheme != "https" {
		httpapi.Error(w, r, httpapi.InvalidInput("url", "the page is not served over HTTPS (after redirects), so mixed content does not apply"))
		return
	}
	res := MixedResult{URL: last.URL, Status: last.Status, ContentType: last.ContentType, BodyBytes: last.BodyBytes, Truncated: last.BodyTruncated, Items: []MixedItem{}}
	if !strings.Contains(strings.ToLower(last.ContentType), "html") {
		httpapi.Error(w, r, httpapi.InvalidInput("url", "the response is not an HTML page ("+last.ContentType+")"))
		return
	}
	for _, v := range headers(last, "Content-Security-Policy") {
		if strings.Contains(strings.ToLower(v), "upgrade-insecure-requests") {
			res.UpgradeInsecureRequests = true
		}
	}
	doc, err := html.Parse(strings.NewReader(last.BodyPreview))
	if err == nil {
		var up bool
		res.Items, res.Resources, up = FindMixed(doc, page)
		res.UpgradeInsecureRequests = res.UpgradeInsecureRequests || up
	}
	if res.Items == nil {
		res.Items = []MixedItem{}
	}
	for _, it := range res.Items {
		switch it.Kind {
		case "active":
			res.Active++
		case "passive":
			res.Passive++
		case "form":
			res.Forms++
		case "link":
			res.Links++
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}
