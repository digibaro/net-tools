// Package tlsscan implements TLS-03: which TLS versions, cipher suites and key-exchange groups
// a server accepts, found by one handshake per option.
package tlsscan

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/stream"
)

// Limits.
const (
	Parallel     = 6
	PerHandshake = 5 * time.Second
	MaxDuration  = 90 * time.Second
)

// Versions scanned, oldest first.
var Versions = []uint16{tls.VersionTLS10, tls.VersionTLS11, tls.VersionTLS12, tls.VersionTLS13}

// Groups offered one at a time with TLS 1.3.
var Groups = []tls.CurveID{tls.X25519MLKEM768, tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521}

// Tool is tls.scan.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "tls.scan" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(MaxDuration+10*time.Second)).Get("/tls/scan/stream", t.handle)
}

// Probe is one handshake attempt (the "probe" event).
type Probe struct {
	Kind      string   `json:"kind"` // version, cipher or group
	Version   string   `json:"version"`
	Name      string   `json:"name"`
	ID        uint16   `json:"id"`
	Supported bool     `json:"supported"`
	Weak      []string `json:"weak"`
}

// Summary is the done payload.
type Summary struct {
	Versions []string            `json:"versions"`
	Ciphers  map[string][]string `json:"ciphers"`
	Groups   []string            `json:"groups"`
	Problems []string            `json:"problems"`
	Warnings []string            `json:"warnings"`
	Rating   string              `json:"rating"` // good, fair or weak
}

// Weaknesses classifies a TLS 1.0–1.2 cipher suite.
func Weaknesses(name string) []string {
	w := []string{}
	switch {
	case strings.Contains(name, "RC4"):
		w = append(w, "rc4")
	case strings.Contains(name, "3DES"):
		w = append(w, "3des")
	}
	if strings.HasPrefix(name, "TLS_RSA_") {
		w = append(w, "no-forward-secrecy")
	}
	if strings.Contains(name, "_CBC_") {
		w = append(w, "cbc")
	}
	return w
}

type scanner struct {
	addr, sni string
}

func (s scanner) try(ctx context.Context, cfg *tls.Config) (tls.ConnectionState, bool) {
	ctx, cancel := context.WithTimeout(ctx, PerHandshake)
	defer cancel()
	d := net.Dialer{}
	raw, err := d.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return tls.ConnectionState{}, false
	}
	defer raw.Close()
	cfg.ServerName, cfg.InsecureSkipVerify = s.sni, true //nolint:gosec // capability scan, not trust
	c := tls.Client(raw, cfg)
	if err := c.HandshakeContext(ctx); err != nil {
		return tls.ConnectionState{}, false
	}
	return c.ConnectionState(), true
}

func suitesFor(v uint16) []*tls.CipherSuite {
	var out []*tls.CipherSuite
	for _, cs := range append(tls.CipherSuites(), tls.InsecureCipherSuites()...) {
		if v != tls.VersionTLS13 && slices.Contains(cs.SupportedVersions, v) {
			out = append(out, cs)
		}
	}
	return out
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	stream.Serve(w, r, func(ctx context.Context) (*stream.Job, error) {
		q := params.New(r.URL.Query())
		host := q.String("host", "", true, 260)
		port := q.Int("port", 443, 1, 65535)
		if err := q.Err(); err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		tgt, err := t.deps.Target.Resolve(ctx, host, target.Opts{AllowPort: true, Field: "host"})
		if err != nil {
			return nil, err //nolint:wrapcheck // client-facing
		}
		if tgt.Port != 0 {
			port = tgt.Port
		}
		s := scanner{addr: tgt.AddrPort(port), sni: tgt.Host}
		if tgt.IsIP {
			s.sni = ""
		}
		meta := map[string]any{"tool": t.ID(), "vantage": "server", "target": map[string]any{"input": host, "address": s.addr}}
		return &stream.Job{Meta: meta, Run: func(ctx context.Context, emit stream.Emit) (any, error) {
			return t.scan(ctx, s, emit), nil
		}}, nil
	})
}

func (t *Tool) scan(ctx context.Context, s scanner, emit stream.Emit) Summary {
	ctx, cancel := context.WithTimeout(ctx, MaxDuration)
	defer cancel()
	var mu sync.Mutex
	send := func(p Probe) {
		if p.Weak == nil {
			p.Weak = []string{}
		}
		mu.Lock()
		emit("probe", p)
		mu.Unlock()
	}
	sum := Summary{Versions: []string{}, Ciphers: map[string][]string{}, Groups: []string{}, Problems: []string{}, Warnings: []string{}}

	// 1. Versions (each with every suite Go can offer for it).
	supported := map[uint16]bool{}
	for _, v := range Versions {
		cfg := &tls.Config{MinVersion: v, MaxVersion: v}
		if v != tls.VersionTLS13 {
			for _, cs := range suitesFor(v) {
				cfg.CipherSuites = append(cfg.CipherSuites, cs.ID)
			}
		}
		_, ok := s.try(ctx, cfg)
		supported[v] = ok
		if ok {
			sum.Versions = append(sum.Versions, tls.VersionName(v))
		}
		send(Probe{Kind: "version", Version: tls.VersionName(v), Name: tls.VersionName(v), ID: v, Supported: ok})
	}

	// 2. Cipher suites per supported version (TLS 1.3 suites are not configurable: report the negotiated one).
	sem := make(chan struct{}, Parallel)
	var wg sync.WaitGroup
	for _, v := range Versions {
		if !supported[v] {
			continue
		}
		if v == tls.VersionTLS13 {
			if st, ok := s.try(ctx, &tls.Config{MinVersion: v, MaxVersion: v}); ok {
				name := tls.CipherSuiteName(st.CipherSuite)
				mu.Lock() // cipher goroutines of earlier versions may still be writing
				sum.Ciphers[tls.VersionName(v)] = []string{name}
				mu.Unlock()
				send(Probe{Kind: "cipher", Version: tls.VersionName(v), Name: name, ID: st.CipherSuite, Supported: true})
			}
			continue
		}
		for _, cs := range suitesFor(v) {
			wg.Go(func() {
				sem <- struct{}{}
				defer func() { <-sem }()
				_, ok := s.try(ctx, &tls.Config{MinVersion: v, MaxVersion: v, CipherSuites: []uint16{cs.ID}}) //nolint:gosec // offering weak suites is the point of the scan
				p := Probe{Kind: "cipher", Version: tls.VersionName(v), Name: cs.Name, ID: cs.ID, Supported: ok, Weak: Weaknesses(cs.Name)}
				if ok {
					mu.Lock()
					sum.Ciphers[p.Version] = append(sum.Ciphers[p.Version], cs.Name)
					mu.Unlock()
				}
				send(p)
			})
		}
	}
	wg.Wait()

	// 3. Key-exchange groups with TLS 1.3 (or 1.2 when that is the newest version).
	gv := uint16(tls.VersionTLS13)
	if !supported[gv] {
		gv = tls.VersionTLS12
	}
	if supported[gv] {
		for _, g := range Groups {
			if g == tls.X25519MLKEM768 && gv != tls.VersionTLS13 {
				continue // the hybrid post-quantum group exists only in TLS 1.3
			}
			wg.Go(func() {
				sem <- struct{}{}
				defer func() { <-sem }()
				_, ok := s.try(ctx, &tls.Config{MinVersion: gv, MaxVersion: gv, CurvePreferences: []tls.CurveID{g}})
				if ok {
					mu.Lock()
					sum.Groups = append(sum.Groups, g.String())
					mu.Unlock()
				}
				send(Probe{Kind: "group", Version: tls.VersionName(gv), Name: g.String(), ID: uint16(g), Supported: ok})
			})
		}
		wg.Wait()
	}
	finish(&sum, supported)
	return sum
}

func finish(sum *Summary, supported map[uint16]bool) {
	for k := range sum.Ciphers {
		slices.Sort(sum.Ciphers[k])
	}
	slices.Sort(sum.Groups)
	if len(sum.Versions) == 0 {
		sum.Problems = append(sum.Problems, "no-tls")
		sum.Rating = "weak"
		return
	}
	if supported[tls.VersionTLS10] || supported[tls.VersionTLS11] {
		sum.Problems = append(sum.Problems, "legacy-versions")
	}
	if !supported[tls.VersionTLS13] {
		sum.Warnings = append(sum.Warnings, "no-tls13")
	}
	weak := map[string]bool{}
	for _, list := range sum.Ciphers {
		for _, c := range list {
			for _, w := range Weaknesses(c) {
				weak[w] = true
			}
		}
	}
	for _, w := range []string{"rc4", "3des", "no-forward-secrecy"} {
		if weak[w] {
			sum.Problems = append(sum.Problems, w)
		}
	}
	if weak["cbc"] {
		sum.Warnings = append(sum.Warnings, "cbc")
	}
	if !slices.Contains(sum.Groups, tls.X25519MLKEM768.String()) && supported[tls.VersionTLS13] {
		sum.Warnings = append(sum.Warnings, "no-post-quantum")
	}
	switch {
	case len(sum.Problems) > 0:
		sum.Rating = "weak"
	case len(sum.Warnings) > 1 || slices.Contains(sum.Warnings, "no-tls13"):
		sum.Rating = "fair"
	default:
		sum.Rating = "good"
	}
}
