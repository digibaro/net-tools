package cloudranges

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestCloudRanges(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path == "/broken" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		b, err := os.ReadFile("testdata" + r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	var sources []Source
	for i, s := range DefaultSources {
		file := []string{"/aws.json", "/gcp.json", "/cloudflare.json", "/fastly.json", "/github.json", "/oracle.json", "/digitalocean.csv"}[i]
		sources = append(sources, Source{Name: s.Name, URL: srv.URL + file, Parse: s.Parse})
	}
	sources = append(sources, Source{Name: "Broken", URL: srv.URL + "/broken", Parse: parseFastly})
	env := tooltest.New(t, &Tool{Sources: sources}, tooltest.Options{AllowPrivate: true})

	cases := map[string]string{
		"52.95.110.1":     "Amazon Web Services|52.95.110.0/24|EC2|us-east-1",
		"34.35.1.1":       "Google Cloud|34.35.0.0/16|Google Cloud|africa-south1",
		"173.245.48.10":   "Cloudflare|173.245.48.0/20|CDN / proxy|",
		"23.235.32.1":     "Fastly|23.235.32.0/20|CDN|",
		"140.82.112.3":    "GitHub|140.82.112.0/20|",
		"40.233.0.1":      "Oracle Cloud|40.233.0.0/19|OCI|mx-monterrey-1",
		"5.101.96.5":      "DigitalOcean|5.101.96.0/21|Droplets|Amsterdam, NL",
		"2606:4700::1111": "Cloudflare|2606:4700::/32|",
	}
	for ip, want := range cases {
		var out Result
		env.Get("/api/v1/cloud-ranges?ip="+ip).Must(t).Decode(t, &out)
		if len(out.Matches) == 0 {
			t.Errorf("%s: no match", ip)
			continue
		}
		m := out.Matches[0]
		got := m.Provider + "|" + m.Prefix + "|" + m.Service + "|" + m.Region
		if !strings.HasPrefix(got, want) {
			t.Errorf("%s: got %s, want %s", ip, got, want)
		}
		if out.Sources[len(out.Sources)-1].Error == "" {
			t.Error("broken source not reported")
		}
	}
	// AWS lists 52.95.110.0/24 inside 52.95.0.0/16: the most specific comes first.
	var out Result
	env.Get("/api/v1/cloud-ranges?ip=52.95.110.1").Must(t).Decode(t, &out)
	if len(out.Matches) != 2 || out.Matches[1].Prefix != "52.95.0.0/16" {
		t.Errorf("nesting = %+v", out.Matches)
	}
	env.Get("/api/v1/cloud-ranges?ip=8.8.8.8").Must(t).Decode(t, &out)
	if out.Provider != "" || len(out.Matches) != 0 {
		t.Errorf("no provider = %+v", out)
	}
	before := hits.Load()
	env.Get("/api/v1/cloud-ranges?ip=1.1.1.1").Must(t)
	if hits.Load() != before+1 { // only the broken source is retried; good lists are cached
		t.Errorf("lists refetched: %d → %d", before, hits.Load())
	}
	env.Get("/api/v1/cloud-ranges?ip=10.0.0.1").ExpectError(t, "INVALID_INPUT")
}
