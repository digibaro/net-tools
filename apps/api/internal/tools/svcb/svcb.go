// Package svcb implements DNS-17: HTTPS / SVCB records (RFC 9460) with decoded ECH configs.
package svcb

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// MaxAliasHops bounds AliasMode chasing.
const MaxAliasHops = 3

// Tool is the HTTPS / SVCB viewer.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.svcb" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(15*time.Second)).Get("/dns/svcb", t.handle)
}

// Param is one SvcParam.
type Param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Suite is an HPKE KDF/AEAD pair.
type Suite struct {
	KDF  string `json:"kdf"`
	AEAD string `json:"aead"`
}

// ECHConfig is one decoded entry of an ECHConfigList (draft-ietf-tls-esni / RFC 9849).
type ECHConfig struct {
	Version       string  `json:"version"`
	ConfigID      int     `json:"configId"`
	KEM           string  `json:"kem"`
	PublicKey     string  `json:"publicKey"`
	Suites        []Suite `json:"suites"`
	MaxNameLength int     `json:"maxNameLength"`
	PublicName    string  `json:"publicName"`
	Extensions    int     `json:"extensions"`
}

// Record is one HTTPS/SVCB record.
type Record struct {
	Owner    string      `json:"owner"`
	Priority int         `json:"priority"`
	Mode     string      `json:"mode"`
	Target   string      `json:"target"`
	TTL      uint32      `json:"ttl"`
	Params   []Param     `json:"params"`
	ECH      []ECHConfig `json:"ech"`
	ECHError string      `json:"echError"`
}

// Result is the response payload.
type Result struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Resolver string   `json:"resolver"`
	Rcode    string   `json:"rcode"`
	Records  []Record `json:"records"`
	// Aliases lists the AliasMode targets that were followed.
	Aliases []string `json:"aliases"`
}

var (
	kems  = map[uint16]string{0x0010: "DHKEM(P-256, HKDF-SHA256)", 0x0011: "DHKEM(P-384, HKDF-SHA384)", 0x0012: "DHKEM(P-521, HKDF-SHA512)", 0x0020: "DHKEM(X25519, HKDF-SHA256)", 0x0021: "DHKEM(X448, HKDF-SHA512)"}
	kdfs  = map[uint16]string{1: "HKDF-SHA256", 2: "HKDF-SHA384", 3: "HKDF-SHA512"}
	aeads = map[uint16]string{1: "AES-128-GCM", 2: "AES-256-GCM", 3: "ChaCha20Poly1305", 0xFFFF: "Export-only"}
)

func name(m map[uint16]string, v uint16) string {
	if s, ok := m[v]; ok {
		return s
	}
	return fmt.Sprintf("0x%04x", v)
}

var errShort = errors.New("ECHConfigList is truncated")

type reader struct{ b []byte }

func (r *reader) u8() (int, error) {
	if len(r.b) < 1 {
		return 0, errShort
	}
	v := r.b[0]
	r.b = r.b[1:]
	return int(v), nil
}

func (r *reader) u16() (uint16, error) {
	if len(r.b) < 2 {
		return 0, errShort
	}
	v := binary.BigEndian.Uint16(r.b)
	r.b = r.b[2:]
	return v, nil
}

func (r *reader) bytes(n int) ([]byte, error) {
	if len(r.b) < n {
		return nil, errShort
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v, nil
}

// ParseECHConfigList decodes the wire-format ECHConfigList of the "ech" SvcParam.
func ParseECHConfigList(b []byte) ([]ECHConfig, error) {
	r := &reader{b}
	total, err := r.u16()
	if err != nil || int(total) != len(r.b) {
		return nil, errors.New("ECHConfigList length does not match")
	}
	var out []ECHConfig
	for len(r.b) > 0 {
		version, err := r.u16()
		if err != nil {
			return out, err
		}
		l, err := r.u16()
		if err != nil {
			return out, err
		}
		body, err := r.bytes(int(l))
		if err != nil {
			return out, err
		}
		if version != 0xfe0d {
			out = append(out, ECHConfig{Version: fmt.Sprintf("0x%04x (unknown)", version), Suites: []Suite{}})
			continue
		}
		c, err := parseContents(body)
		if err != nil {
			return out, err
		}
		out = append(out, c)
	}
	return out, nil
}

func parseContents(b []byte) (ECHConfig, error) {
	r := &reader{b}
	c := ECHConfig{Version: "0xfe0d", Suites: []Suite{}}
	id, err := r.u8()
	if err != nil {
		return c, err
	}
	c.ConfigID = id
	kem, err := r.u16()
	if err != nil {
		return c, err
	}
	c.KEM = name(kems, kem)
	pkLen, err := r.u16()
	if err != nil {
		return c, err
	}
	pk, err := r.bytes(int(pkLen))
	if err != nil {
		return c, err
	}
	c.PublicKey = hex.EncodeToString(pk)
	csLen, err := r.u16()
	if err != nil {
		return c, err
	}
	cs, err := r.bytes(int(csLen))
	if err != nil {
		return c, err
	}
	for i := 0; i+4 <= len(cs); i += 4 {
		c.Suites = append(c.Suites, Suite{KDF: name(kdfs, binary.BigEndian.Uint16(cs[i:])), AEAD: name(aeads, binary.BigEndian.Uint16(cs[i+2:]))})
	}
	if c.MaxNameLength, err = r.u8(); err != nil {
		return c, err
	}
	nl, err := r.u8()
	if err != nil {
		return c, err
	}
	pn, err := r.bytes(nl)
	if err != nil {
		return c, err
	}
	c.PublicName = string(pn)
	extLen, err := r.u16()
	if err != nil {
		return c, err
	}
	ext, err := r.bytes(int(extLen))
	if err != nil {
		return c, err
	}
	for er := (&reader{ext}); len(er.b) >= 4; {
		_, _ = er.u16()
		n, _ := er.u16()
		if _, err := er.bytes(int(n)); err != nil {
			break
		}
		c.Extensions++
	}
	return c, nil
}

// FromRR converts an HTTPS or SVCB resource record.
func FromRR(rr dns.RR) (Record, bool) {
	var s *dns.SVCB
	switch v := rr.(type) {
	case *dns.HTTPS:
		s = &v.SVCB
	case *dns.SVCB:
		s = v
	default:
		return Record{}, false
	}
	rec := Record{Owner: strings.TrimSuffix(s.Hdr.Name, "."), Priority: int(s.Priority), Target: s.Target, TTL: s.Hdr.Ttl, Params: []Param{}, ECH: []ECHConfig{}, Mode: "service"}
	if s.Priority == 0 {
		rec.Mode = "alias"
	}
	for _, kv := range s.Value {
		rec.Params = append(rec.Params, Param{Key: kv.Key().String(), Value: kv.String()})
		if e, ok := kv.(*dns.SVCBECHConfig); ok {
			cfgs, err := ParseECHConfigList(e.ECH)
			rec.ECH = append(rec.ECH, cfgs...)
			if err != nil {
				rec.ECHError = err.Error()
			}
		}
	}
	return rec, true
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("name", "", true, 253)
	typ := strings.ToUpper(q.String("type", "HTTPS", false, 5))
	resolver := q.String("resolver", "system", false, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	qtype := map[string]uint16{"HTTPS": dns.TypeHTTPS, "SVCB": dns.TypeSVCB}[typ]
	if qtype == 0 {
		httpapi.Error(w, r, httpapi.InvalidInput("type", "type must be HTTPS or SVCB"))
		return
	}
	n := strings.TrimSpace(raw)
	if !strings.HasPrefix(n, "_") { // _dns.resolver.arpa, _443._https.example are valid owner names
		var err error
		if n, err = target.NormalizeHostname(n); err != nil {
			httpapi.Error(w, r, httpapi.InvalidInput("name", "not a valid domain name"))
			return
		}
	} else if _, ok := dns.IsDomainName(n); !ok {
		httpapi.Error(w, r, httpapi.InvalidInput("name", "not a valid domain name"))
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, resolver, dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	start := time.Now()
	res := Result{Name: strings.TrimSuffix(n, "."), Type: typ, Resolver: srv.Label, Records: []Record{}, Aliases: []string{}}
	cur := n
	for hop := 0; hop <= MaxAliasHops; hop++ {
		msg, _, err := dnsx.Query(r.Context(), srv, cur, qtype)
		if err != nil {
			httpapi.Error(w, r, dnslookup.Upstream(err))
			return
		}
		if hop == 0 {
			res.Rcode = dnsx.RcodeName(msg.Rcode)
		}
		next := ""
		for _, rr := range msg.Answer {
			if rec, ok := FromRR(rr); ok {
				res.Records = append(res.Records, rec)
				if rec.Mode == "alias" && rec.Target != "." && next == "" {
					next = rec.Target
				}
			}
		}
		if next == "" || hop == MaxAliasHops {
			break
		}
		res.Aliases = append(res.Aliases, strings.TrimSuffix(next, "."))
		cur = next
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", DurationMs: dnslookup.Ms(time.Since(start)), Target: &httpapi.TargetMeta{Input: raw, Used: srv.Addr.String()}})
}
