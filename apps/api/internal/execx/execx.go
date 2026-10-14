// Package execx runs allow-listed binaries without a shell (docs/SECURITY.md §3): absolute paths
// resolved at startup, argv arrays only, minimal environment, no stdin, bounded output, and the
// whole process group killed when the context ends.
package execx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode"
)

// MaxOutput bounds stdout read per run; MaxStderr bounds captured stderr.
const (
	MaxOutput = 1 << 20
	MaxStderr = 64 << 10
)

// ErrNotAllowed is returned for binaries that are not on the allow-list or not installed.
var ErrNotAllowed = errors.New("binary not available")

// Runner executes allow-listed binaries.
type Runner struct {
	paths map[string]string
}

// New resolves each allow-listed name to the first existing absolute candidate path.
// Missing binaries are simply unavailable (tools report binary_missing).
func New(candidates map[string][]string) *Runner {
	r := &Runner{paths: map[string]string{}}
	for name, list := range candidates {
		for _, p := range list {
			if !strings.HasPrefix(p, "/") {
				continue
			}
			if found, err := exec.LookPath(p); err == nil {
				r.paths[name] = found
				break
			}
		}
	}
	return r
}

// DefaultCandidates is the production allow-list.
func DefaultCandidates() map[string][]string {
	return map[string][]string{
		"mtr":        {"/usr/bin/mtr", "/usr/sbin/mtr"},
		"traceroute": {"/usr/bin/traceroute", "/usr/sbin/traceroute"},
		"iperf3":     {"/usr/bin/iperf3"},
		"bgpq4":      {"/usr/bin/bgpq4"},
	}
}

// Available reports whether name can be run.
func (r *Runner) Available(name string) bool {
	_, ok := r.paths[name]
	return ok
}

// UserValue validates a user-supplied argument: it must not look like a flag and must not
// contain whitespace or control characters. Builders call this for every user-derived value.
func UserValue(s string) (string, error) {
	if s == "" || strings.HasPrefix(s, "-") {
		return "", fmt.Errorf("invalid argument %q", s)
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", fmt.Errorf("invalid argument %q", s)
		}
	}
	return s, nil
}

// Run executes name with args, calling onLine for each stdout line. It returns captured stderr.
// Output beyond MaxOutput is discarded and the process is killed.
func (r *Runner) Run(ctx context.Context, name string, args []string, onLine func(string)) (string, error) {
	path, ok := r.paths[name]
	if !ok {
		return "", fmt.Errorf("%s: %w", name, ErrNotAllowed)
	}
	cmd := exec.CommandContext(ctx, path, args...) //nolint:gosec // path from allow-list, argv from builders
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C"}
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Kill the whole process group (mtr spawns mtr-packet).
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = 2 * time.Second

	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, n: MaxStderr}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s: start: %w", name, err)
	}

	sc := bufio.NewScanner(io.LimitReader(stdout, MaxOutput))
	sc.Buffer(make([]byte, 64<<10), 64<<10)
	for sc.Scan() {
		if onLine != nil {
			onLine(sc.Text())
		}
	}
	_, _ = io.Copy(io.Discard, stdout) // drain so Wait can finish if the limit was hit
	err = cmd.Wait()
	if ctx.Err() != nil {
		return stderr.String(), ctx.Err() //nolint:wrapcheck // propagate cancellation/deadline as-is
	}
	if err != nil {
		return stderr.String(), fmt.Errorf("%s: %w", name, err)
	}
	return stderr.String(), nil
}

type limitedWriter struct {
	w io.Writer
	n int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n <= 0 {
		return len(p), nil
	}
	q := p
	if len(q) > l.n {
		q = q[:l.n]
	}
	l.n -= len(q)
	if _, err := l.w.Write(q); err != nil {
		return 0, err //nolint:wrapcheck // in-memory buffer
	}
	return len(p), nil
}
