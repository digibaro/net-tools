package peeringdb

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func TestPeeringDB(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		file := "testdata/net-empty.json"
		if r.URL.Query().Get("asn") == "13335" && r.URL.Query().Get("depth") == "2" {
			file = "testdata/net-13335.json"
		}
		b, _ := os.ReadFile(file)
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	env := tooltest.New(t, &Tool{Base: srv.URL}, tooltest.Options{AllowPrivate: true, Env: map[string]string{"PEERINGDB_KEY": "k"}})
	var out Network
	env.Get("/api/v1/peeringdb/AS13335").Must(t).Decode(t, &out)
	if !out.Found || out.Name != "Cloudflare" || out.Policy != "Open" || out.IRRASSet != "AS13335:AS-CLOUDFLARE" || out.IXCount != 5 || out.IXs[0].SpeedMb < out.IXs[1].SpeedMb || !out.IXs[1].RSPeer || out.IXs[1].IPv6 != "" || len(out.Facilities) != 3 || out.PeeringDBURL != "https://www.peeringdb.com/asn/13335" {
		t.Errorf("out = %+v", out)
	}
	if gotAuth != "Api-Key k" {
		t.Errorf("auth = %q", gotAuth)
	}
	env.Get("/api/v1/peeringdb/64512").Must(t).Decode(t, &out)
	if out.Found || out.ASN != 64512 {
		t.Errorf("missing = %+v", out)
	}
	env.Get("/api/v1/peeringdb/nope").ExpectError(t, "INVALID_INPUT")
}
