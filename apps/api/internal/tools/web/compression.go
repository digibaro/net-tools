package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/httpcore"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Encodings are requested one at a time; the body is read raw (not decoded) to measure transfer size.
var Encodings = []string{"identity", "gzip", "br", "zstd", "deflate"}

// CompressionTool is WEB-07.
type CompressionTool struct{ deps tools.Deps }

func (*CompressionTool) ID() string                                { return "web.compression" }
func (*CompressionTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *CompressionTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(60*time.Second)).Get("/http/compression", t.handle)
}

// EncodingResult is one Accept-Encoding attempt.
type EncodingResult struct {
	Requested string  `json:"requested"`
	Served    string  `json:"served"`
	Supported bool    `json:"supported"`
	Bytes     int     `json:"bytes"`
	Truncated bool    `json:"truncated"`
	Ratio     float64 `json:"ratio"`
	Error     string  `json:"error"`
}

// CompressionResult is the WEB-07 payload.
type CompressionResult struct {
	URL         string           `json:"url"`
	ContentType string           `json:"contentType"`
	Vary        string           `json:"vary"`
	Original    int              `json:"original"`
	Results     []EncodingResult `json:"results"`
	Best        string           `json:"best"`
}

func (t *CompressionTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	u := q.String("url", "", true, 2048)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := CompressionResult{URL: u, Results: []EncodingResult{}}
	final := u
	for i, enc := range Encodings {
		hops, err := fetch(r.Context(), t.deps, httpcore.Request{URL: final, FollowRedirects: i == 0, Headers: map[string]string{"Accept-Encoding": enc}})
		if err != nil {
			if i == 0 {
				httpapi.Error(w, r, err) // the URL itself is unusable (policy, syntax, DNS)
				return
			}
			out.Results = append(out.Results, EncodingResult{Requested: enc, Error: httpapi.ToAPIError(err).Message})
			continue
		}
		last := hops[len(hops)-1]
		if i == 0 {
			final = last.URL // later requests go straight to the final URL
			out.URL, out.ContentType = last.URL, last.ContentType
			out.Vary, _ = header(last, "Vary")
		}
		served, _ := header(last, "Content-Encoding")
		served = strings.ToLower(strings.TrimSpace(served))
		if served == "" {
			served = "identity"
		}
		res := EncodingResult{Requested: enc, Served: served, Bytes: last.BodyBytes, Truncated: last.BodyTruncated, Error: last.Error}
		res.Supported = enc != "identity" && served == enc
		if i == 0 && served == "identity" {
			out.Original = last.BodyBytes
		}
		out.Results = append(out.Results, res)
	}
	best := 0
	for i := range out.Results {
		res := &out.Results[i]
		if out.Original > 0 && res.Error == "" {
			res.Ratio = float64(int(float64(res.Bytes)/float64(out.Original)*1000)) / 1000
		}
		if res.Supported && (out.Best == "" || res.Bytes < best) {
			out.Best, best = res.Requested, res.Bytes
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: u}})
}
