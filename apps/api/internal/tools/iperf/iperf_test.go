package iperf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/execx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeIperf replays a captured --json-stream fixture and records argv.
func fakeIperf(t *testing.T, fixture string, exit int) (*execx.Runner, string) {
	t.Helper()
	dir := t.TempDir()
	argsFile, script := filepath.Join(dir, "args"), filepath.Join(dir, "iperf3")
	abs, _ := filepath.Abs(filepath.Join("testdata", fixture))
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\ncat " + abs + "\nexit " + strconv.Itoa(exit) + "\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // test fixture
		t.Fatal(err)
	}
	return execx.New(map[string][]string{"iperf3": {script}}), argsFile
}

func done(t *testing.T, body string) Summary {
	t.Helper()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("no done: %s", body)
	}
	var d struct {
		Summary Summary `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	return d.Summary
}

func TestIperfTCP(t *testing.T) {
	ex, argsFile := fakeIperf(t, "tcp.jsonl", 0)
	env := tooltest.New(t, &Tool{}, tooltest.Options{Exec: ex, Lookup: tooltest.StaticLookup("93.184.215.14")})
	body := env.Get("/api/v1/iperf/stream?server=iperf.example.com&duration=2&parallel=2&reverse=true").Raw.Body.String()
	s := done(t, body)
	if s.Protocol != "TCP" || s.SentMbps < 12000 || s.ReceivedMbps < 12000 || s.Retransmits == nil || *s.Retransmits != 1 || s.JitterMs != nil || strings.Count(body, "event: interval") != 2 {
		t.Errorf("summary = %+v", s)
	}
	args, _ := os.ReadFile(argsFile)
	if got := strings.Fields(string(args)); strings.Join(got, " ") != "-c 93.184.215.14 -p 5201 -t 2 -i 1 -P 2 --connect-timeout 3000 --json-stream -R" {
		t.Errorf("argv = %v", got)
	}
}

func TestIperfUDP(t *testing.T) {
	ex, argsFile := fakeIperf(t, "udp.jsonl", 0)
	env := tooltest.New(t, &Tool{}, tooltest.Options{Exec: ex, Lookup: tooltest.StaticLookup("93.184.215.14")})
	s := done(t, env.Get("/api/v1/iperf/stream?server=iperf.example.com&proto=udp&bitrate=10&duration=2").Raw.Body.String())
	if s.Protocol != "UDP" || s.JitterMs == nil || *s.JitterMs <= 0 || s.LostPercent == nil || *s.LostPackets != 0 || s.SentMbps < 9.9 {
		t.Errorf("summary = %+v", s)
	}
	if args, _ := os.ReadFile(argsFile); !strings.Contains(string(args), "-u\n-b\n10M") {
		t.Errorf("argv = %s", args)
	}
}

func TestIperfErrorsAndPolicy(t *testing.T) {
	ex, _ := fakeIperf(t, "err.jsonl", 1)
	env := tooltest.New(t, &Tool{}, tooltest.Options{Exec: ex, Lookup: tooltest.StaticLookup("93.184.215.14")})
	if b := env.Get("/api/v1/iperf/stream?server=iperf.example.com&duration=2").Raw.Body.String(); !strings.Contains(b, "Connection refused") || !strings.Contains(b, "PROVIDER_ERROR") {
		t.Errorf("error = %s", b)
	}
	for q, want := range map[string]string{"server=10.0.0.5": "TARGET_PRIVATE_DENIED", "server=169.254.169.254": "TARGET_BLOCKED", "server=x.example&duration=60": "INVALID_INPUT", "server=x.example&parallel=9": "INVALID_INPUT", "server=-x": "INVALID_INPUT"} {
		if b := env.Get("/api/v1/iperf/stream?" + q).Raw.Body.String(); !strings.Contains(b, want) {
			t.Errorf("%s: %s", q, b)
		}
	}
	if ok, reason := (&Tool{}).Enabled(tools.Deps{Config: tooltest.Config(t, nil), Exec: execx.New(nil)}); ok || reason != tools.ReasonBinaryMissing {
		t.Errorf("no binary: %v %s", ok, reason)
	}
	if ok, reason := (&Tool{}).Enabled(tools.Deps{Config: tooltest.Config(t, map[string]string{"IPERF_ENABLED": "false"}), Exec: ex}); ok || reason != tools.ReasonDisabledByAdmin {
		t.Errorf("flag off: %v %s", ok, reason)
	}
}
