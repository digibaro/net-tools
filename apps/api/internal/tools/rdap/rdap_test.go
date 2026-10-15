package rdap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func fakeRDAP(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		switch r.URL.Path {
		case "/bootstrap/dns.json":
			_ = json.NewEncoder(w).Encode(map[string]any{"version": "1.0", "publication": "2026-01-01T00:00:00Z", "services": [][]any{{[]string{"test"}, []string{srv.URL + "/rdap/"}}}})
		case "/rdap/domain/example.test":
			_, _ = w.Write([]byte(`{"objectClassName":"domain","handle":"D1","ldhName":"EXAMPLE.TEST","status":["client transfer prohibited"],
			"events":[{"eventAction":"registration","eventDate":"1995-08-14T04:00:00Z"},{"eventAction":"expiration","eventDate":"2030-08-13T04:00:00Z"}],
			"nameservers":[{"objectClassName":"nameserver","ldhName":"NS1.EXAMPLE.TEST"}],
			"entities":[{"objectClassName":"entity","handle":"292","roles":["registrar"],"vcardArray":["vcard",[["version",{},"text","4.0"],["fn",{},"text","Example Registrar, Inc."]]]}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRDAPDomain(t *testing.T) {
	srv := fakeRDAP(t)
	svc := &Service{BootstrapURL: srv.URL + "/bootstrap/"}
	env := tooltest.New(t, &LookupTool{Svc: svc}, tooltest.Options{AllowPrivate: true})
	var rec Record
	env.Get("/api/v1/rdap?query=example.test").Must(t).Decode(t, &rec)
	if rec.Name != "example.test" || rec.Registrar != "Example Registrar, Inc." || rec.Nameservers[0] != "ns1.example.test" || len(rec.Events) != 2 || !strings.HasSuffix(rec.Server, "/rdap/domain/example.test") {
		t.Errorf("rec = %+v", rec)
	}
	env.Get("/api/v1/rdap?query=missing.test").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/rdap?query=example.nobootstrap").ExpectError(t, "PROVIDER_ERROR")

	exp := tooltest.New(t, &ExpiryTool{Svc: svc, Now: func() time.Time { return time.Date(2030, 8, 3, 4, 0, 0, 0, time.UTC) }}, tooltest.Options{AllowPrivate: true})
	var e Expiry
	exp.Get("/api/v1/domain/expiry?domain=example.test").Must(t).Decode(t, &e)
	if e.DaysLeft == nil || *e.DaysLeft != 10 || e.Registered != "1995-08-14T04:00:00Z" || e.Registrar != "Example Registrar, Inc." {
		t.Errorf("expiry = %+v", e)
	}
	exp.Get("/api/v1/domain/expiry?domain=8.8.8.8").ExpectError(t, "INVALID_INPUT")
}

// The bootstrap may point anywhere; the policy client must refuse private RDAP servers.
func TestRDAPPolicy(t *testing.T) {
	srv := fakeRDAP(t)
	svc := &Service{BootstrapURL: srv.URL + "/bootstrap/"}
	env := tooltest.New(t, &LookupTool{Svc: svc}, tooltest.Options{})
	r := env.Get("/api/v1/rdap?query=example.test")
	if r.OK || (r.Error.Code != "TARGET_PRIVATE_DENIED" && r.Error.Code != "PROVIDER_ERROR") {
		t.Fatalf("got %d %s", r.Status, r.Error.Code)
	}
}
