package portscan

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestParsePorts(t *testing.T) {
	if p, _ := ParsePorts("web", 100); !slices.Contains(p, 443) {
		t.Error(p)
	}
	if p, err := ParsePorts("443, 22,80-82,22", 100); err != nil || !slices.Equal(p, []int{22, 80, 81, 82, 443}) {
		t.Error(p, err)
	}
	for _, bad := range []string{"0", "70000", "90-80", "a", "", "1-2000"} {
		if _, err := ParsePorts(bad, 1024); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func fakeDial(_ context.Context, a netip.AddrPort) error {
	switch a.Port() {
	case 22, 443:
		return nil
	case 25:
		return errors.New("i/o timeout")
	}
	return syscall.ECONNREFUSED
}

func TestPortscan(t *testing.T) {
	env := tooltest.New(t, &Tool{Dial: fakeDial}, tooltest.Options{Lookup: tooltest.StaticLookup("93.184.215.14"), Env: map[string]string{"PORTSCAN_ENABLED": "true", "PORTSCAN_RATE_PPS": "10000"}})
	body := env.Get("/api/v1/portscan/stream?target=example.com&ports=20-30,443").Raw.Body.String()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("body = %s", body)
	}
	var d struct {
		Summary Summary `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	s := d.Summary
	if s.Scanned != 12 || len(s.Open) != 2 || s.Open[0].Service != "ssh" || s.Filtered != 1 || s.Closed != 9 || !s.Complete || strings.Count(body, "event: port") != 2 {
		t.Errorf("summary = %+v", s)
	}
}

func TestPortscanPolicyAndFlag(t *testing.T) {
	ok, reason := (&Tool{}).Enabled(tools.Deps{Config: tooltest.Config(t, map[string]string{"NETTOOLS_PROFILE": "public"})})
	if ok || reason != tools.ReasonDisabledByAdmin {
		t.Errorf("public profile: %v %s", ok, reason)
	}
	env := tooltest.New(t, &Tool{Dial: fakeDial}, tooltest.Options{Env: map[string]string{"PORTSCAN_ENABLED": "true"}})
	for q, want := range map[string]string{"target=10.0.0.1": "TARGET_PRIVATE_DENIED", "target=169.254.169.254": "TARGET_BLOCKED", "target=example.com&ports=1-5000": "INVALID_INPUT"} {
		if b := env.Get("/api/v1/portscan/stream?" + q).Raw.Body.String(); !strings.Contains(b, want) {
			t.Errorf("%s: %s", q, b)
		}
	}
}
