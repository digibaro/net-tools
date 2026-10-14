package web

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// RedirectsTool is WEB-02: follow and show every redirect hop.
type RedirectsTool struct{ deps tools.Deps }

func (*RedirectsTool) ID() string                                { return "web.redirects" }
func (*RedirectsTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *RedirectsTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(45*time.Second)).Get("/http/redirects", t.handle)
}

func (t *RedirectsTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	u := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: u, Method: "GET", FollowRedirects: true})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	httpapi.JSON(w, r, HopsResult{Hops: hops}, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: u}})
}

// TimingTool is WEB-03: DNS / connect / TLS / TTFB / transfer breakdown of one request.
type TimingTool struct{ deps tools.Deps }

func (*TimingTool) ID() string                                { return "web.timing" }
func (*TimingTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *TimingTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(30*time.Second)).Get("/http/timing", t.handle)
}

func (t *TimingTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	u := q.String("url", "", true, 2048)
	follow := q.Bool("follow", false)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: u, Method: "GET", FollowRedirects: follow})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	httpapi.JSON(w, r, HopsResult{Hops: hops}, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: u}})
}
