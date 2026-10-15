// Package sshaudit implements TLS-10: an SSH server's banner, host keys and offered algorithms,
// without authenticating.
package sshaudit

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/ssh"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// PerConnection bounds each connection; maxPacket the KEXINIT size we accept.
const (
	PerConnection = 8 * time.Second
	maxPacket     = 35000
)

// HostKeyTypes are requested one at a time to collect every host key.
var HostKeyTypes = []string{ssh.KeyAlgoED25519, ssh.KeyAlgoECDSA256, ssh.KeyAlgoECDSA384, ssh.KeyAlgoECDSA521, ssh.KeyAlgoRSASHA512}

// Tool is ssh.audit.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "ssh.audit" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(40*time.Second)).Get("/ssh/audit", t.handle)
}

// KexInit is the server's algorithm offer (RFC 4253 §7.1).
type KexInit struct {
	Kex         []string `json:"kex"`
	HostKey     []string `json:"hostKey"`
	Ciphers     []string `json:"ciphers"`
	MACs        []string `json:"macs"`
	Compression []string `json:"compression"`
}

// HostKey is one host key with its fingerprint.
type HostKey struct {
	Type        string `json:"type"`
	Bits        int    `json:"bits"`
	Fingerprint string `json:"fingerprint"`
}

// Finding flags an algorithm.
type Finding struct {
	Algorithm string `json:"algorithm"`
	Kind      string `json:"kind"`
	Severity  string `json:"severity"` // fail or warn
	Reason    string `json:"reason"`
}

// Result is the TLS-10 payload.
type Result struct {
	Address  string    `json:"address"`
	Banner   string    `json:"banner"`
	Software string    `json:"software"`
	Algos    KexInit   `json:"algorithms"`
	HostKeys []HostKey `json:"hostKeys"`
	Findings []Finding `json:"findings"`
	Terrapin bool      `json:"terrapin"`
}

func readPacket(r *bufio.Reader) ([]byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("read packet: %w", err)
	}
	length := binary.BigEndian.Uint32(hdr[:4])
	pad := int(hdr[4])
	if length < uint32(hdr[4])+1 || length > maxPacket {
		return nil, errors.New("invalid packet length")
	}
	body := make([]byte, length-1)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("read packet: %w", err)
	}
	return body[:len(body)-pad], nil
}

// ParseKexInit decodes an SSH_MSG_KEXINIT payload.
func ParseKexInit(p []byte) (KexInit, error) {
	if len(p) < 17 || p[0] != 20 {
		return KexInit{}, errors.New("not a KEXINIT message")
	}
	p = p[17:] // message id + 16-byte cookie
	lists := make([][]string, 10)
	for i := range lists {
		if len(p) < 4 {
			return KexInit{}, errors.New("truncated KEXINIT")
		}
		n := binary.BigEndian.Uint32(p)
		if int64(n) > int64(len(p)-4) {
			return KexInit{}, errors.New("truncated KEXINIT")
		}
		if n > 0 {
			lists[i] = strings.Split(string(p[4:4+n]), ",")
		} else {
			lists[i] = []string{}
		}
		p = p[4+n:]
	}
	return KexInit{Kex: lists[0], HostKey: lists[1], Ciphers: union(lists[2], lists[3]), MACs: union(lists[4], lists[5]), Compression: union(lists[6], lists[7])}, nil
}

func union(a, b []string) []string {
	out := slices.Clone(a)
	for _, x := range b {
		if !slices.Contains(out, x) {
			out = append(out, x)
		}
	}
	return out
}

// kexinit connects, exchanges banners and reads the server's KEXINIT.
func kexinit(ctx context.Context, addr string) (string, KexInit, error) {
	ctx, cancel := context.WithTimeout(ctx, PerConnection)
	defer cancel()
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", KexInit{}, fmt.Errorf("connect: %w", err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(PerConnection))
	r := bufio.NewReader(io.LimitReader(c, 64<<10))
	var banner string
	for range 20 { // servers may send lines before the identification string (RFC 4253 §4.2)
		l, err := r.ReadString('\n')
		if err != nil {
			return "", KexInit{}, errors.New("no SSH banner")
		}
		if strings.HasPrefix(l, "SSH-") {
			banner = strings.TrimRight(l, "\r\n")
			break
		}
	}
	if banner == "" {
		return "", KexInit{}, errors.New("no SSH banner")
	}
	if _, err := fmt.Fprint(c, "SSH-2.0-NetTools_audit\r\n"); err != nil {
		return banner, KexInit{}, fmt.Errorf("write: %w", err)
	}
	p, err := readPacket(r)
	if err != nil {
		return banner, KexInit{}, err
	}
	k, err := ParseKexInit(p)
	return banner, k, err
}

var errGotKey = errors.New("host key received")

// hostKey completes key exchange for one host key algorithm and stops before authentication.
func hostKey(ctx context.Context, addr, algo string) (*HostKey, error) {
	ctx, cancel := context.WithTimeout(ctx, PerConnection)
	defer cancel()
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(PerConnection))
	var got *HostKey
	cfg := &ssh.ClientConfig{
		User: "nettools-audit", HostKeyAlgorithms: []string{algo}, Timeout: PerConnection,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			got = &HostKey{Type: key.Type(), Fingerprint: ssh.FingerprintSHA256(key), Bits: keyBits(key)}
			return errGotKey
		},
	}
	_, _, _, err = ssh.NewClientConn(c, addr, cfg)
	if got != nil {
		return got, nil
	}
	return nil, fmt.Errorf("handshake: %w", err)
}

func keyBits(k ssh.PublicKey) int {
	if ck, ok := k.(ssh.CryptoPublicKey); ok {
		switch pk := ck.CryptoPublicKey().(type) {
		case interface{ Size() int }:
			return pk.Size() * 8 // *rsa.PublicKey
		}
	}
	switch k.Type() {
	case ssh.KeyAlgoED25519:
		return 256
	case ssh.KeyAlgoECDSA256:
		return 256
	case ssh.KeyAlgoECDSA384:
		return 384
	case ssh.KeyAlgoECDSA521:
		return 521
	}
	return 0
}

var weak = []struct {
	match             func(string) bool
	kind, sev, reason string
}{
	{func(a string) bool { return a == "diffie-hellman-group1-sha1" }, "kex", "fail", "1024-bit group with SHA-1 (Logjam)"},
	{func(a string) bool { return strings.HasSuffix(a, "-sha1") && strings.HasPrefix(a, "diffie-hellman") }, "kex", "warn", "uses SHA-1"},
	{func(a string) bool { return a == "ssh-dss" }, "hostKey", "fail", "DSA host keys are broken and removed from OpenSSH"},
	{func(a string) bool { return a == "ssh-rsa" }, "hostKey", "warn", "RSA signatures with SHA-1 (prefer rsa-sha2-256/512)"},
	{func(a string) bool {
		return strings.HasPrefix(a, "arcfour") || a == "3des-cbc" || strings.HasPrefix(a, "blowfish") || a == "cast128-cbc"
	}, "cipher", "fail", "obsolete cipher"},
	{func(a string) bool { return strings.HasSuffix(a, "-cbc") }, "cipher", "warn", "CBC mode (prefer CTR, GCM or ChaCha20-Poly1305)"},
	{func(a string) bool {
		return strings.Contains(a, "md5") || strings.HasPrefix(a, "umac-64") || a == "hmac-sha1-96"
	}, "mac", "fail", "weak MAC"},
	{func(a string) bool { return a == "hmac-sha1" || a == "hmac-sha1-etm@openssh.com" }, "mac", "warn", "SHA-1 MAC"},
}

// Analyze flags weak algorithms and Terrapin exposure (CVE-2023-48795).
func Analyze(k KexInit) ([]Finding, bool) {
	out := []Finding{}
	check := func(kind string, list []string) {
		for _, a := range list {
			for _, w := range weak {
				if w.kind == kind && w.match(a) {
					out = append(out, Finding{Algorithm: a, Kind: kind, Severity: w.sev, Reason: w.reason})
					break
				}
			}
		}
	}
	check("kex", k.Kex)
	check("hostKey", k.HostKey)
	check("cipher", k.Ciphers)
	check("mac", k.MACs)
	pq := slices.ContainsFunc(k.Kex, func(a string) bool { return strings.Contains(a, "sntrup") || strings.Contains(a, "mlkem") })
	if !pq {
		out = append(out, Finding{Kind: "kex", Severity: "warn", Reason: "no post-quantum key exchange (mlkem768x25519-sha256 or sntrup761x25519-sha512)"})
	}
	strict := slices.Contains(k.Kex, "kex-strict-s-v00@openssh.com")
	chacha := slices.Contains(k.Ciphers, "chacha20-poly1305@openssh.com")
	etmCBC := slices.ContainsFunc(k.MACs, func(m string) bool { return strings.Contains(m, "-etm@") }) && slices.ContainsFunc(k.Ciphers, func(c string) bool { return strings.HasSuffix(c, "-cbc") })
	terrapin := !strict && (chacha || etmCBC)
	if terrapin {
		out = append(out, Finding{Kind: "kex", Severity: "fail", Reason: "vulnerable to Terrapin (CVE-2023-48795): ChaCha20-Poly1305 or EtM with CBC offered without strict key exchange"})
	}
	return out, terrapin
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	host := q.String("host", "", true, 260)
	port := q.Int("port", 22, 1, 65535)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	tgt, err := t.deps.Target.Resolve(ctx, host, target.Opts{AllowPort: true, Field: "host"})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if tgt.Port != 0 {
		port = tgt.Port
	}
	addr := tgt.AddrPort(port)
	banner, k, err := kexinit(ctx, addr)
	if err != nil {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInvalidInput, Message: "no SSH server answered: " + err.Error(), Details: map[string]any{"field": "host"}, Err: err})
		return
	}
	out := Result{Address: addr, Banner: banner, Software: strings.TrimPrefix(strings.SplitN(banner, " ", 2)[0], "SSH-2.0-"), Algos: k, HostKeys: []HostKey{}}
	out.Findings, out.Terrapin = Analyze(k)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, algo := range HostKeyTypes {
		if !slices.Contains(k.HostKey, algo) {
			continue
		}
		wg.Go(func() {
			if hk, err := hostKey(ctx, addr, algo); err == nil {
				mu.Lock()
				out.HostKeys = append(out.HostKeys, *hk)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	slices.SortFunc(out.HostKeys, func(a, b HostKey) int { return strings.Compare(a.Type, b.Type) })
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: host, Used: addr}})
}
