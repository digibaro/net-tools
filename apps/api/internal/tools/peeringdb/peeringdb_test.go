package peeringdb

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
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

func TestCommon(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := "testdata/net-empty.json"
		switch r.URL.Query().Get("asn") {
		case "13335":
			file = "testdata/net-13335.json"
		case "64500":
			file = "testdata/net-64500.json"
		}
		b, _ := os.ReadFile(file)
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	env := tooltest.New(t, &CommonTool{Base: srv.URL}, tooltest.Options{AllowPrivate: true})
	var out Common
	env.Get("/api/v1/peeringdb/common?a=AS13335&b=64500").Must(t).Decode(t, &out)
	if !out.A.Found || !out.B.Found || out.B.Policy != "Selective" || len(out.IXs) != 2 || len(out.Facilities) != 1 || out.Facilities[0].FacID != 4 {
		t.Fatalf("out = %+v", out)
	}
	byID := map[int]CommonIX{}
	for _, x := range out.IXs {
		byID[x.IXID] = x
	}
	if !byID[31].BothRS || byID[255].BothRS || len(byID[255].A) != 2 || byID[31].B[0].IPv4 != "80.81.192.5" {
		t.Errorf("ixs = %+v", out.IXs)
	}
	env.Get("/api/v1/peeringdb/common?a=AS13335&b=64512").Must(t).Decode(t, &out)
	if out.B.Found || len(out.IXs) != 0 {
		t.Errorf("missing network = %+v", out)
	}
	env.Get("/api/v1/peeringdb/common?a=x&b=1").ExpectError(t, "INVALID_INPUT")
}
