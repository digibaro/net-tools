package irr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/execx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeBgpq4 records argv and replays a captured fixture: -t → asns, AS-EMPTY → empty, AS-FAIL → exit 1.
func fakeBgpq4(t *testing.T) (*execx.Runner, string) {
	t.Helper()
	dir := t.TempDir()
	td, _ := filepath.Abs("testdata")
	argsFile, script := filepath.Join(dir, "args"), filepath.Join(dir, "bgpq4")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\n" +
		"for a in \"$@\"; do case \"$a\" in AS-FAIL) echo 'FATAL: Unable to connect to rr.ntt.net' >&2; exit 1;; AS-EMPTY) cat " + td + "/empty.json; exit 0;; -t) cat " + td + "/asns.json; exit 0;; esac; done\n" +
		"cat " + td + "/prefixes4.json\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // test fixture
		t.Fatal(err)
	}
	return execx.New(map[string][]string{"bgpq4": {script}}), argsFile
}

func TestIRR(t *testing.T) {
	ex, argsFile := fakeBgpq4(t)
	env := tooltest.New(t, &Tool{}, tooltest.Options{Exec: ex})
	argv := func() string { b, _ := os.ReadFile(argsFile); return strings.Join(strings.Fields(string(b)), " ") }

	var res Result
	env.Get("/api/v1/bgp/irr?set=as13335&sources=ripe,RADB,ripe").Must(t).Decode(t, &res)
	if res.Set != "AS13335" || len(res.Prefixes) != 12 || res.Prefixes[0].Prefix != "1.0.0.0/24" || !res.Prefixes[0].Exact || res.Family != 4 || !res.Aggregate {
		t.Errorf("prefixes = %+v", res)
	}
	var ranged *Prefix
	for i := range res.Prefixes {
		if !res.Prefixes[i].Exact {
			ranged = &res.Prefixes[i]
		}
	}
	if ranged == nil || ranged.GreaterEqual != 24 || ranged.LessEqual != 24 {
		t.Errorf("ranged = %+v", ranged)
	}
	if got := argv(); got != "-j -S RIPE,RADB -l prefixes -4 -A AS13335" {
		t.Errorf("argv = %s", got)
	}

	env.Get("/api/v1/bgp/irr?set=AS-CLOUDFLARE&mode=asns").Must(t).Decode(t, &res)
	if len(res.ASNs) != 1692 || res.ASNs[0] != 26 || len(res.Prefixes) != 0 || res.Aggregate {
		t.Errorf("asns = %d %+v", len(res.ASNs), res.Prefixes)
	}
	if got := argv(); got != "-j -l asns -t AS-CLOUDFLARE" {
		t.Errorf("argv = %s", got)
	}

	env.Get("/api/v1/bgp/irr?set=AS-EMPTY&family=6&aggregate=false").Must(t).Decode(t, &res)
	if len(res.Prefixes) != 0 || res.Family != 6 {
		t.Errorf("empty = %+v", res)
	}
	if got := argv(); got != "-j -l prefixes -6 AS-EMPTY" {
		t.Errorf("argv = %s", got)
	}

	if r := env.Get("/api/v1/bgp/irr?set=AS-FAIL"); r.OK || r.Error.Code != "PROVIDER_ERROR" || !strings.Contains(r.Error.Message, "Unable to connect") {
		t.Errorf("fail = %+v", r.Error)
	}
	for _, q := range []string{"set=-h evil", "set=AS-X;rm", "set=FOO", "set=AS-X&sources=EVIL", "set=AS-X&mode=routes"} {
		env.Get("/api/v1/bgp/irr?"+strings.ReplaceAll(q, " ", "%20")).ExpectError(t, "INVALID_INPUT")
	}
}

func TestValidSet(t *testing.T) {
	for s, want := range map[string]bool{"AS65000": true, "AS-EXAMPLE": true, "as65000:as-customers": true, "AS-A:AS-B": true, "RS-ROUTES": true, "AS65000:RS-X": true, "-hAS": false, "AS": false, "AS-": false, "AS-X Y": false} {
		if ValidSet(s) != want {
			t.Errorf("%q", s)
		}
	}
	if ok, reason := (&Tool{}).Enabled(tools.Deps{Exec: execx.New(nil)}); ok || reason != tools.ReasonBinaryMissing {
		t.Errorf("%v %s", ok, reason)
	}
}
