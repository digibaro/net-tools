// Package speedtest implements CON-09: download/upload/latency between the browser and this
// server. Disabled in the public profile (SPEEDTEST_ENABLED) because it serves bulk traffic.
package speedtest

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Limits per request; the browser runs a few parallel streams.
const (
	MaxDownload     = 100 << 20
	DefaultDownload = 25 << 20
	MaxUpload       = 32 << 20
	chunk           = 1 << 20
)

var (
	noiseOnce sync.Once
	noise     []byte
)

// randomChunk is generated once: incompressible so proxies cannot shrink the transfer.
func randomChunk() []byte {
	noiseOnce.Do(func() {
		noise = make([]byte, chunk)
		_, _ = rand.Read(noise)
	})
	return noise
}

// Tool is net.speedtest.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string { return "net.speedtest" }

// Enabled requires SPEEDTEST_ENABLED (default on outside the public profile).
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	if !d.Config.Features.Speedtest {
		return false, tools.ReasonDisabledByAdmin
	}
	return true, ""
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	bulk := r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(60*time.Second))
	bulk.Get("/speedtest/download", t.download)
	bulk.Post("/speedtest/upload", t.upload)
	r.With(d.RateLimit(limits.Light)).Get("/speedtest/ping", t.ping)
}

func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no") // nginx: stream, do not buffer
}

func (t *Tool) download(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	n := q.Int("bytes", DefaultDownload, chunk, MaxDownload)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	noStore(w)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(n))
	buf := randomChunk()
	for left := n; left > 0; left -= len(buf) {
		if _, err := w.Write(buf[:min(left, len(buf))]); err != nil {
			return // client stopped the test
		}
	}
}

// UploadResult is the upload payload.
type UploadResult struct {
	Bytes int64   `json:"bytes"`
	Ms    float64 `json:"ms"`
}

func (t *Tool) upload(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	body := http.MaxBytesReader(w, r.Body, MaxUpload)
	n, err := io.Copy(io.Discard, body)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			httpapi.Error(w, r, httpapi.InvalidInput("body", "uploads are limited to 32 MiB per request"))
			return
		}
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInvalidInput, Message: "upload interrupted", Err: err})
		return
	}
	noStore(w)
	httpapi.JSON(w, r, UploadResult{Bytes: n, Ms: float64(time.Since(start).Microseconds()) / 1000}, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}

func (t *Tool) ping(w http.ResponseWriter, _ *http.Request) {
	noStore(w)
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("pong"))
}
