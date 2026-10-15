package torcheck

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestTorExit(t *testing.T) {
	var listHits atomic.Int32
	onionooDown := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/list":
			listHits.Add(1)
			b, _ := os.ReadFile("testdata/torbulkexitlist.txt")
			_, _ = w.Write(b)
		case "/details":
			if onionooDown {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			file := "testdata/onionoo-empty.json"
			if r.URL.Query().Get("search") == "192.42.116.44" {
				file = "testdata/onionoo-192.42.116.44.json"
			}
			b, _ := os.ReadFile(file)
			_, _ = w.Write(b)
		}
	}))
	defer srv.Close()
	env := tooltest.New(t, &Tool{ListURL: srv.URL + "/list", OnionooBase: srv.URL}, tooltest.Options{AllowPrivate: true})
	var out Result
	env.Get("/api/v1/tor-exit?ip=192.42.116.44").Must(t).Decode(t, &out)
	if !out.ExitNode || out.ListSize != 20 || len(out.Relays) != 2 || out.Relays[0].Nickname != "NTH44R8" || !out.Relays[0].Exit || out.Relays[0].Country != "NL" {
		t.Errorf("out = %+v", out)
	}
	env.Get("/api/v1/tor-exit?ip=8.8.8.8").Must(t).Decode(t, &out)
	if out.ExitNode || len(out.Relays) != 0 {
		t.Errorf("non-exit = %+v", out)
	}
	if listHits.Load() != 1 {
		t.Errorf("exit list fetched %d times, want 1 (cached)", listHits.Load())
	}
	onionooDown = true
	env.Get("/api/v1/tor-exit?ip=171.25.193.25").Must(t).Decode(t, &out)
	if !out.ExitNode || out.RelayError == "" {
		t.Errorf("onionoo down = %+v", out)
	}
	env.Get("/api/v1/tor-exit?ip=10.1.1.1").ExpectError(t, "INVALID_INPUT")
}
