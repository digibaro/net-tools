package execx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func script(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "tool.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o700); err != nil { //nolint:gosec // test fixture
		t.Fatal(err)
	}
	return p
}

func TestRunStreamsLinesWithMinimalEnv(t *testing.T) {
	r := New(map[string][]string{"tool": {script(t, `echo "arg=$1"; echo "path=$PATH"; echo "secret=$SECRET"; echo oops >&2`)}})
	t.Setenv("SECRET", "leak")
	var lines []string
	stderr, err := r.Run(context.Background(), "tool", []string{"x; rm -rf /"}, func(s string) { lines = append(lines, s) })
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"arg=x; rm -rf /", "path=/usr/sbin:/usr/bin:/sbin:/bin", "secret="}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q", lines)
	}
	if stderr != "oops\n" {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestRunNotAllowed(t *testing.T) {
	r := New(map[string][]string{"relative": {"sh"}, "missing": {"/nonexistent/bin"}})
	for _, name := range []string{"relative", "missing", "sh"} {
		if _, err := r.Run(context.Background(), name, nil, nil); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("%s: err = %v", name, err)
		}
		if r.Available(name) {
			t.Errorf("%s must not be available", name)
		}
	}
}

func TestRunKillsProcessGroupOnCancel(t *testing.T) {
	r := New(map[string][]string{"tool": {script(t, "sleep 30 & sleep 30; echo never")}})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := r.Run(ctx, "tool", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("took %s; child processes were not killed", d)
	}
}

func TestRunBoundsOutput(t *testing.T) {
	r := New(map[string][]string{"tool": {script(t, "yes line | head -c 3000000")}})
	n := 0
	if _, err := r.Run(context.Background(), "tool", nil, func(string) { n++ }); err != nil {
		t.Fatal(err)
	}
	if n*5 > MaxOutput+5 {
		t.Errorf("read %d lines, more than MaxOutput allows", n)
	}
}

func TestUserValue(t *testing.T) {
	for _, ok := range []string{"example.com", "192.0.2.1", "2001:db8::1"} {
		if _, err := UserValue(ok); err != nil {
			t.Errorf("%q rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "-oProxyCommand=x", "--help", "a b", "a\nb", "a\tb", "\x00"} {
		if _, err := UserValue(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
