package latmatrix

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/tools/ping"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

// fakeEngine answers with RTT = last octet ms; .99 never answers.
func fakeEngine(_ context.Context, ip netip.Addr, o ping.Options, onReply func(ping.Reply, time.Duration)) (int, error) {
	last := ip.As4()[3]
	for i := range o.Count {
		if last == 99 {
			continue
		}
		d := time.Duration(last) * time.Millisecond
		onReply(ping.Reply{Seq: i, RTTMs: float64(last)}, d)
	}
	return o.Count, nil
}

func TestLatencyMatrix(t *testing.T) {
	lookup := func(_ context.Context, _, host string) ([]netip.Addr, error) {
		ip, ok := map[string]string{"far.test": "93.0.0.80", "near.test": "93.0.0.5", "dead.test": "93.0.0.99", "inside.test": "10.0.0.1"}[strings.TrimSuffix(host, ".")]
		if !ok {
			return nil, errors.New("no such host " + host)
		}
		return []netip.Addr{netip.MustParseAddr(ip)}, nil
	}
	env := tooltest.New(t, &Tool{Engine: fakeEngine}, tooltest.Options{Lookup: lookup})
	body := env.Get("/api/v1/latency-matrix/stream?targets=far.test,near.test,dead.test,inside.test&count=3").Raw.Body.String()
	if strings.Count(body, "event: row") != 4 {
		t.Fatalf("body = %s", body)
	}
	i := strings.Index(body, "event: done\ndata: ")
	var d struct {
		Summary Summary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d); err != nil {
		t.Fatal(err)
	}
	rows := d.Summary.Rows
	if rows[0].Target != "near.test" || rows[1].Target != "far.test" || rows[0].Stats.AvgMs != 5 || *rows[0].RTTs[2] != 5 {
		t.Errorf("order = %+v", rows)
	}
	var inside Row
	for _, r := range rows {
		if r.Target == "inside.test" {
			inside = r
		}
	}
	if !strings.Contains(inside.Error, "private") || inside.Address != "" {
		t.Errorf("policy row = %+v", inside)
	}
	for q, want := range map[string]string{"targets=": "INVALID_INPUT", "targets=a1.test,a2.test,a3.test,a4.test,a5.test,a6.test,a7.test,a8.test,a9.test,a10.test,a11.test": "INVALID_INPUT", "targets=a.test&count=500": "INVALID_INPUT"} {
		if b := env.Get("/api/v1/latency-matrix/stream?" + q).Raw.Body.String(); !strings.Contains(b, want) {
			t.Errorf("%s: %s", q, b)
		}
	}
}
