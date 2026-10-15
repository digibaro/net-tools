package remote

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeGlobalping replays testdata/<type>.json: the first poll reports every probe but the first
// as still running, the second poll returns the finished measurement.
func fakeGlobalping(t *testing.T) (*httptest.Server, *atomic.Value) {
	t.Helper()
	var lastAuth atomic.Value
	lastAuth.Store("")
	var polls atomic.Int32
	var typ atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastAuth.Store(r.Header.Get("Authorization"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/measurements":
			var body struct {
				Type      string `json:"type"`
				Locations []struct {
					Magic string `json:"magic"`
				} `json:"locations"`
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			if len(body.Locations) > 0 && body.Locations[0].Magic == "Atlantis" {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"error":{"type":"no_probes_found","message":"No suitable probes found."}}`))
				return
			}
			typ.Store(body.Type)
			polls.Store(0)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"id":"m1","probesCount":2}`))
		case r.URL.Path == "/measurements/m1":
			data, err := os.ReadFile("testdata/" + typ.Load().(string) + ".json")
			if err != nil {
				t.Error(err)
				return
			}
			if polls.Add(1) == 1 {
				var m map[string]any
				_ = json.Unmarshal(data, &m)
				m["status"] = "in-progress"
				results := m["results"].([]any)
				for _, res := range results[1:] {
					res.(map[string]any)["result"] = map[string]any{"status": "in-progress"}
				}
				data, _ = json.Marshal(m)
			}
			_, _ = w.Write(data)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &lastAuth
}

func events(body, name string) []string {
	var out []string
	for _, block := range strings.Split(body, "\n\n") {
		if strings.HasPrefix(block, "event: "+name+"\n") {
			out = append(out, strings.TrimPrefix(block, "event: "+name+"\ndata: "))
		}
	}
	return out
}

func TestRemoteStream(t *testing.T) {
	srv, auth := fakeGlobalping(t)
	env := tooltest.New(t, &Tool{BaseURL: srv.URL, PollEvery: time.Millisecond}, tooltest.Options{AllowPrivate: true, Env: map[string]string{"GLOBALPING_TOKEN": "tok"}})
	body := env.Get("/api/v1/remote/stream?type=ping&target=example.com&locations=Europe&limit=2").Raw.Body.String()
	probes := events(body, "probe")
	if len(probes) != 2 || len(events(body, "done")) != 1 || !strings.Contains(body, `"vantage":"globalping"`) {
		t.Fatalf("body = %s", body)
	}
	var first Result
	if err := json.Unmarshal([]byte(probes[0]), &first); err != nil || first.Probe.City != "Falkenstein" || first.Status != "finished" || !strings.Contains(string(first.Result), `"avg":5.666`) {
		t.Errorf("first = %+v %v", first, err)
	}
	if auth.Load() != "Bearer tok" {
		t.Errorf("auth = %v", auth.Load())
	}
	for _, typ := range []string{"traceroute", "mtr", "dns", "http"} {
		body := env.Get("/api/v1/remote/stream?type=" + typ + "&target=example.com&limit=1").Raw.Body.String()
		if len(events(body, "probe")) == 0 || strings.Contains(body, `"rawHeaders"`) {
			t.Errorf("%s: %s", typ, body)
		}
	}
}

func TestRemoteErrors(t *testing.T) {
	srv, _ := fakeGlobalping(t)
	env := tooltest.New(t, &Tool{BaseURL: srv.URL, PollEvery: time.Millisecond}, tooltest.Options{AllowPrivate: true})
	for q, want := range map[string]string{
		"type=ping&target=169.254.169.254":                    "TARGET_BLOCKED",
		"type=ping&target=10.0.0.1":                           "TARGET_PRIVATE_DENIED",
		"type=ping&target=[::ffff:127.0.0.1]":                 "TARGET_PRIVATE_DENIED",
		"type=ping&target=localhost":                          "INVALID_INPUT",
		"type=dns&target=1.1.1.1":                             "INVALID_INPUT",
		"type=portscan&target=example.com":                    "INVALID_INPUT",
		"type=ping&target=example.com&limit=50":               "INVALID_INPUT",
		"type=ping&target=example.com&locations=a,b,c,d,e,f":  "INVALID_INPUT",
		"type=ping&target=example.com&locations=%3Cscript%3E": "INVALID_INPUT",
		"type=ping&target=example.com&locations=Atlantis":     "No suitable probes found.",
	} {
		body := env.Get("/api/v1/remote/stream?" + q).Raw.Body.String()
		if !strings.Contains(body, "event: error") || !strings.Contains(body, want) {
			t.Errorf("%s: %s", q, body)
		}
	}
}
