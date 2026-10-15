package pmtu

import (
	"context"
	"encoding/json"
	"net/netip"
	"strconv"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// prober simulating a path MTU of mtu (IPv4); 0 = no ICMP replies at all.
func pathOf(mtu int) Prober {
	return func(_ context.Context, ip netip.Addr, payload int) bool {
		over := overhead4
		if ip.Is6() {
			over = overhead6
		}
		return mtu > 0 && payload+over <= mtu
	}
}

func done(t *testing.T, body string) Result {
	t.Helper()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("no done: %s", body)
	}
	var d struct {
		Summary Result `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	return d.Summary
}

func TestPMTU(t *testing.T) {
	for _, tc := range []struct {
		path, max, want, mss int
	}{{1492, 1500, 1492, 1452}, {1500, 1500, 1500, 1460}, {1420, 1500, 1420, 1380}, {9000, 9000, 9000, 8960}} {
		env := tooltest.New(t, &Tool{Prober: pathOf(tc.path)}, tooltest.Options{Lookup: tooltest.StaticLookup("93.184.215.14")})
		r := done(t, env.Get("/api/v1/pmtu/stream?target=example.com&max="+strconv.Itoa(tc.max)).Raw.Body.String())
		if r.PathMTU != tc.want || r.MSS != tc.mss || !r.Reachable || r.Probes > 14 {
			t.Errorf("path %d: %+v", tc.path, r)
		}
	}
	env := tooltest.New(t, &Tool{Prober: pathOf(0)}, tooltest.Options{Lookup: tooltest.StaticLookup("93.184.215.14")})
	if r := done(t, env.Get("/api/v1/pmtu/stream?target=example.com").Raw.Body.String()); r.Reachable || r.Note == "" {
		t.Errorf("unreachable = %+v", r)
	}
	v6 := tooltest.New(t, &Tool{Prober: pathOf(1480)}, tooltest.Options{Lookup: tooltest.StaticLookup("2606:2800:21f:cb07:6820:80da:af6b:8b2c")})
	if r := done(t, v6.Get("/api/v1/pmtu/stream?target=example.com").Raw.Body.String()); r.PathMTU != 1480 || r.MSS != 1420 {
		t.Errorf("v6 = %+v", r)
	}
	strict := tooltest.New(t, &Tool{Prober: pathOf(1500)}, tooltest.Options{})
	if b := strict.Get("/api/v1/pmtu/stream?target=192.168.1.1").Raw.Body.String(); !strings.Contains(b, "TARGET_PRIVATE_DENIED") {
		t.Error(b)
	}
	if b := strict.Get("/api/v1/pmtu/stream?target=example.com&max=20000").Raw.Body.String(); !strings.Contains(b, "INVALID_INPUT") {
		t.Error(b)
	}
}
