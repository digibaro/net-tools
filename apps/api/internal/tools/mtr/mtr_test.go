package mtr

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/execx"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

const rawOutput = `x 0 1
h 0 192.0.2.1
d 0 gw.example.
p 0 1500 1
x 1 2
x 2 3
h 2 93.184.215.14
p 2 12345 3
x 0 4
p 0 2500 4
x 1 5
x 2 6
p 2 11000 6
x 3 7
`

func TestParser(t *testing.T) {
	p := NewParser()
	for _, l := range strings.Split(rawOutput, "\n") {
		p.Line(l)
	}
	hops := p.Hops()
	if len(hops) != 3 {
		t.Fatalf("hops = %+v", hops)
	}
	h1, h2, h3 := hops[0], hops[1], hops[2]
	if h1.IP != "192.0.2.1" || h1.Host != "gw.example." || h1.Sent != 2 || h1.Recv != 2 || h1.Loss != 0 || h1.BestMs != 1.5 || h1.Worst != 2.5 || h1.AvgMs != 2 {
		t.Errorf("hop1 = %+v", h1)
	}
	if h2.IP != "" || h2.Sent != 2 || h2.Recv != 0 || h2.Loss != 100 {
		t.Errorf("hop2 = %+v", h2)
	}
	if h3.IP != "93.184.215.14" || !slices.Equal(h3.RTTMs, []float64{12.345, 11}) || h3.LastMs != 11 {
		t.Errorf("hop3 = %+v", h3)
	}
	if p.Line("garbage") != 0 || p.Line("p x 1 2") != 0 || p.Line("p 999 1 1") != 0 {
		t.Error("bad lines must be ignored")
	}
}

func TestArgs(t *testing.T) {
	args, err := Args("93.184.215.14", "tcp", 443, 3, 30, 200*time.Millisecond)
	want := []string{"--raw", "--report-cycles", "3", "--interval", "0.2", "--max-ttl", "30", "--gracetime", "2", "--tcp", "--port", "443", "--", "93.184.215.14"}
	if err != nil || !slices.Equal(args, want) {
		t.Errorf("args = %q %v", args, err)
	}
	if _, err := Args("-oFoo", "icmp", 0, 1, 1, time.Second); err == nil {
		t.Error("flag-like target accepted")
	}
}

func fakeMtr(t *testing.T, out string) (*execx.Runner, string) {
	t.Helper()
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	script := filepath.Join(dir, "mtr")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\ncat <<'EOF'\n" + out + "EOF\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // test fixture
		t.Fatal(err)
	}
	return execx.New(map[string][]string{"mtr": {script}}), argsFile
}

func fakeASN(_ context.Context, name string) ([]string, error) {
	switch name {
	case "14.215.184.93.origin.asn.cymru.com":
		return []string{"15133 | 93.184.215.0/24 | US | ripencc | 2008-06-02"}, nil
	case "AS15133.asn.cymru.com":
		return []string{"15133 | US | arin | 2001-06-01 | EDGECAST, US"}, nil
	}
	return nil, nil
}

func TestTracerouteStream(t *testing.T) {
	runner, argsFile := fakeMtr(t, rawOutput)
	env := tooltest.New(t, &Tool{Mode: Traceroute, ASN: fakeASN}, tooltest.Options{Exec: runner, Lookup: tooltest.StaticLookup("93.184.215.14")})
	body := env.Get("/api/v1/traceroute/stream?target=example.com&proto=udp&maxHops=20").Raw.Body.String()
	if !strings.Contains(body, "event: hop") || !strings.Contains(body, `"hops":[`) || !strings.Contains(body, `"host":"gw.example."`) || !strings.Contains(body, `"asn":15133`) {
		t.Fatalf("body = %s", body)
	}
	args, _ := os.ReadFile(argsFile)
	if !strings.Contains(string(args), "--udp\n--\n93.184.215.14\n") || strings.Contains(string(args), "example.com") {
		t.Errorf("mtr must get the pinned IP only: %q", args)
	}
}

func TestMTRLimitsAndSSRF(t *testing.T) {
	runner, _ := fakeMtr(t, rawOutput)
	env := tooltest.New(t, &Tool{Mode: MTR, UpdateEvery: time.Nanosecond, ASN: fakeASN}, tooltest.Options{Exec: runner, Lookup: tooltest.StaticLookup("172.16.5.5"), Env: map[string]string{"MTR_MAX_CYCLES": "5", "TRACEROUTE_MAX_HOPS": "20"}})
	for q, code := range map[string]string{
		"target=8.8.8.8&cycles=6":   "INVALID_INPUT",
		"target=8.8.8.8&maxHops=21": "INVALID_INPUT",
		"target=8.8.8.8&proto=sctp": "INVALID_INPUT",
		"target=10.0.0.1":           "TARGET_PRIVATE_DENIED",
		"target=nas.example":        "TARGET_PRIVATE_DENIED",
		"target=169.254.169.254":    "TARGET_BLOCKED",
		"target=-oProxy":            "INVALID_INPUT",
	} {
		body := env.Get("/api/v1/mtr/stream?" + q).Raw.Body.String()
		if !strings.Contains(body, `"code":"`+code+`"`) {
			t.Errorf("%s: %s", q, body)
		}
	}
	body := env.Get("/api/v1/mtr/stream?target=8.8.8.8&cycles=2").Raw.Body.String()
	if strings.Count(body, "event: hop") < 3 {
		t.Errorf("mtr updates: %s", body)
	}
}

func TestBinaryMissing(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Exec: execx.New(nil)})
	r := env.Get("/api/v1/capabilities")
	if !strings.Contains(r.Raw.Body.String(), `"net.traceroute":{"enabled":false,"reason":"binary_missing"}`) {
		t.Errorf("capabilities = %s", r.Raw.Body.String())
	}
}
