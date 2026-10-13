package ping

import (
	"context"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func fake(lost map[int]bool) (Engine, *Options, *netip.Addr) {
	var got Options
	var ip netip.Addr
	return func(_ context.Context, addr netip.Addr, o Options, onReply func(Reply, time.Duration)) (int, error) {
		got, ip = o, addr
		for i := range o.Count {
			if !lost[i] {
				onReply(Reply{Seq: i, RTTMs: float64(10 + i), TTL: 57, From: addr.String()}, time.Duration(10+i)*time.Millisecond)
			}
		}
		return o.Count, nil
	}, &got, &ip
}

func TestPingStream(t *testing.T) {
	engine, opts, ip := fake(map[int]bool{1: true})
	env := tooltest.New(t, &Tool{Engine: engine}, tooltest.Options{Lookup: tooltest.StaticLookup("93.184.215.14")})
	body := env.Get("/api/v1/ping/stream?target=example.com&count=3&interval=250&size=100").Raw.Body.String()
	if strings.Count(body, "event: reply") != 2 || !strings.Contains(body, `"lossPct":33.333`) || !strings.Contains(body, "event: done") {
		t.Fatalf("body = %s", body)
	}
	if opts.Count != 3 || opts.Interval != 250*time.Millisecond || opts.Size != 100 || ip.String() != "93.184.215.14" {
		t.Errorf("engine got %+v %s", opts, ip)
	}
	if !strings.HasPrefix(body, "event: meta") || !strings.Contains(body, `"used":"93.184.215.14"`) {
		t.Errorf("meta missing: %s", body)
	}
}

func TestPingLimits(t *testing.T) {
	engine, _, _ := fake(nil)
	env := tooltest.New(t, &Tool{Engine: engine}, tooltest.Options{Env: map[string]string{"PING_MAX_COUNT": "4", "PING_MIN_INTERVAL_MS": "500"}})
	for q, code := range map[string]string{
		"target=8.8.8.8&count=5":      "INVALID_INPUT",
		"target=8.8.8.8&count=0":      "INVALID_INPUT",
		"target=8.8.8.8&interval=100": "INVALID_INPUT",
		"target=8.8.8.8&size=9000":    "INVALID_INPUT",
		"target=":                     "INVALID_INPUT",
	} {
		body := env.Get("/api/v1/ping/stream?" + q).Raw.Body.String()
		if !strings.Contains(body, `"code":"`+code+`"`) {
			t.Errorf("%s: %s", q, body)
		}
	}
}

func TestPingSSRF(t *testing.T) {
	engine, _, _ := fake(nil)
	env := tooltest.New(t, &Tool{Engine: engine}, tooltest.Options{Lookup: tooltest.StaticLookup("10.0.0.5")})
	for target, code := range map[string]string{
		"127.0.0.1":          "TARGET_PRIVATE_DENIED",
		"::1":                "TARGET_PRIVATE_DENIED",
		"10.1.1.1":           "TARGET_PRIVATE_DENIED",
		"169.254.169.254":    "TARGET_BLOCKED",
		"::ffff:192.168.0.1": "TARGET_PRIVATE_DENIED",
		"0x7f.1":             "INVALID_INPUT",
		"2130706433":         "INVALID_INPUT",
		"internal.example":   "TARGET_PRIVATE_DENIED",
		"224.0.0.1":          "TARGET_BLOCKED",
	} {
		body := env.Get("/api/v1/ping/stream?target=" + target).Raw.Body.String()
		if !strings.Contains(body, `"code":"`+code+`"`) || strings.Contains(body, "event: reply") {
			t.Errorf("%s: %s", target, body)
		}
	}
}
