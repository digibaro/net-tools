package web

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestCompression(t *testing.T) {
	page := strings.Repeat("<p>hello compression</p>\n", 400)
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write([]byte(page))
	_ = zw.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/page", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Vary", "Accept-Encoding")
		if r.Header.Get("Accept-Encoding") == "gzip" {
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write(gz.Bytes())
			return
		}
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()
	var out CompressionResult
	tooltest.New(t, &CompressionTool{}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/http/compression?url="+url.QueryEscape(srv.URL+"/start")).Must(t).Decode(t, &out)
	if !strings.HasSuffix(out.URL, "/page") || out.Original != len(page) || out.Best != "gzip" || out.Vary != "Accept-Encoding" {
		t.Fatalf("out = %+v", out)
	}
	for _, r := range out.Results {
		switch r.Requested {
		case "gzip":
			if !r.Supported || r.Bytes != gz.Len() || r.Ratio > 0.1 {
				t.Errorf("gzip = %+v", r)
			}
		case "br", "zstd":
			if r.Supported || r.Served != "identity" {
				t.Errorf("%s = %+v", r.Requested, r)
			}
		}
	}
	tooltest.New(t, &CompressionTool{}, tooltest.Options{}).Get("/api/v1/http/compression?url="+url.QueryEscape("http://127.0.0.1/")).ExpectError(t, "TARGET_PRIVATE_DENIED")
}
