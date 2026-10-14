package observatory

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func TestObservatory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/scan" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Query().Get("host") {
		case "github.com":
			b, _ := os.ReadFile("testdata/scan-github.com.json")
			_, _ = w.Write(b)
		case "down.example":
			_, _ = w.Write([]byte(`{"id":1,"grade":null,"error":"site down","score":null}`))
		default:
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"invalid-hostname","message":"Invalid hostname"}`))
		}
	}))
	defer srv.Close()
	env := tooltest.New(t, &Tool{Base: srv.URL}, tooltest.Options{AllowPrivate: true})
	var out Result
	env.Get("/api/v1/http/observatory?host=github.com").Must(t).Decode(t, &out)
	if out.Grade != "A+" || out.Score != 115 || out.TestsPassed != 11 || out.DetailsURL == "" {
		t.Errorf("out = %+v", out)
	}
	if r := env.Get("/api/v1/http/observatory?host=down.example"); r.OK || r.Error.Message != "Mozilla Observatory: site down" {
		t.Errorf("down = %+v", r.Error)
	}
	if r := env.Get("/api/v1/http/observatory?host=bad.example"); r.OK || r.Error.Message != "Mozilla Observatory: Invalid hostname" {
		t.Errorf("422 = %+v", r.Error)
	}
	env.Get("/api/v1/http/observatory?host=localhost").ExpectError(t, "INVALID_INPUT")
}
