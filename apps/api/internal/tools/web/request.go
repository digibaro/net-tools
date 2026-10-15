package web

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/httpcore"
)

// RequestTool is WEB-01: send an arbitrary request and inspect the response.
type RequestTool struct{ deps tools.Deps }

func (*RequestTool) ID() string                                { return "web.request" }
func (*RequestTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *RequestTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(30*time.Second)).Post("/http/request", t.handle)
}

// RequestBody is the JSON input of /http/request.
type RequestBody struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	FollowRedirects bool              `json:"followRedirects"`
}

// HopsResult wraps a request chain.
type HopsResult struct {
	Hops []httpcore.Hop `json:"hops"`
}

func (t *RequestTool) handle(w http.ResponseWriter, r *http.Request) {
	var in RequestBody
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, httpcore.MaxRequestBody+16<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("body", "expected JSON {url, method, headers, body, followRedirects}"))
		return
	}
	if in.Method == "" {
		in.Method = "GET"
	}
	if !slices.Contains(httpcore.Methods, in.Method) {
		httpapi.Error(w, r, httpapi.InvalidInput("method", "unsupported method"))
		return
	}
	if len(in.Body) > httpcore.MaxRequestBody {
		httpapi.Error(w, r, httpapi.InvalidInput("body", "request body is limited to 64 KiB"))
		return
	}
	if err := httpcore.ValidateHeaders(in.Headers); err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("headers", err.Error()))
		return
	}
	hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: in.URL, Method: in.Method, Headers: in.Headers, Body: in.Body, FollowRedirects: in.FollowRedirects, PreviewBytes: 64 << 10})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	httpapi.JSON(w, r, HopsResult{Hops: hops}, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: in.URL, Used: hops[0].RemoteAddr}})
}
