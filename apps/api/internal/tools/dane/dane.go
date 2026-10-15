// Package dane implements TLS-12: DANE / TLSA check (RFC 6698, 7671, 7672) for SMTP and HTTPS.
package dane

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tlsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// MaxHosts bounds the MX hosts checked for a mail domain.
const MaxHosts = 4

// Tool is the DANE checker.
type Tool struct {
	deps tools.Deps
	// Port overrides the service port when dialing (tests); the TLSA name keeps the real port.
	Port int
}

func (*Tool) ID() string                                { return "tls.dane" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(60*time.Second)).Get("/tls/dane", t.handle)
}

// TLSA is one record and whether it matches the presented chain.
type TLSA struct {
	Usage        int    `json:"usage"`
	Selector     int    `json:"selector"`
	MatchingType int    `json:"matchingType"`
	Data         string `json:"data"`
	Match        bool   `json:"match"`
	// MatchedCert is the chain index that matched (0 = leaf), -1 for none.
	MatchedCert int `json:"matchedCert"`
}

// Host is the result for one server name.
type Host struct {
	Name     string   `json:"name"`
	TLSAName string   `json:"tlsaName"`
	Rcode    string   `json:"rcode"`
	DNSSEC   bool     `json:"dnssec"`
	Records  []TLSA   `json:"records"`
	Address  string   `json:"address"`
	TLS      *Summary `json:"tls"`
	Error    string   `json:"error"`
	// Verdict: valid, mismatch, insecure (records without DNSSEC), no-tlsa, unreachable.
	Verdict string `json:"verdict"`
}

// Summary is the part of the TLS session the page shows.
type Summary struct {
	Version     string      `json:"version"`
	Cipher      string      `json:"cipher"`
	Chain       []tlsx.Cert `json:"chain"`
	Verified    bool        `json:"verified"`
	VerifyError string      `json:"verifyError"`
}

// Result is the response payload.
type Result struct {
	Input    string `json:"input"`
	Service  string `json:"service"`
	Port     int    `json:"port"`
	Resolver string `json:"resolver"`
	MX       bool   `json:"mx"`
	Hosts    []Host `json:"hosts"`
}

// Matches reports whether a TLSA record matches cert (selector and matching type applied).
func Matches(rec TLSA, cert *x509.Certificate) bool {
	var data []byte
	switch rec.Selector {
	case 0:
		data = cert.Raw
	case 1:
		data = cert.RawSubjectPublicKeyInfo
	default:
		return false
	}
	switch rec.MatchingType {
	case 1:
		s := sha256.Sum256(data)
		data = s[:]
	case 2:
		s := sha512.Sum512(data)
		data = s[:]
	case 0:
	default:
		return false
	}
	want, err := hex.DecodeString(rec.Data)
	return err == nil && bytes.Equal(want, data)
}

// MatchChain fills Match/MatchedCert: end-entity usages (1, 3) match the leaf only, trust-anchor
// usages (0, 2) any certificate of the presented chain.
func MatchChain(recs []TLSA, chain []*x509.Certificate) {
	for i := range recs {
		recs[i].MatchedCert = -1
		for j, c := range chain {
			if (recs[i].Usage == 1 || recs[i].Usage == 3) && j > 0 {
				break
			}
			if Matches(recs[i], c) {
				recs[i].Match, recs[i].MatchedCert = true, j
				break
			}
		}
	}
}

// Verdict rates a host (RFC 7672: TLSA without DNSSEC must be ignored).
func Verdict(h Host) string {
	switch {
	case h.Error != "" && h.TLS == nil && len(h.Records) == 0:
		return "unreachable"
	case len(h.Records) == 0:
		return "no-tlsa"
	case !h.DNSSEC:
		return "insecure"
	case h.TLS == nil: // signed TLSA, but no certificate to compare (e.g. port 25 blocked)
		return "unreachable"
	}
	for _, r := range h.Records {
		// PKIX usages also need a valid WebPKI chain.
		if r.Match && (r.Usage >= 2 || (h.TLS != nil && h.TLS.Verified)) {
			return "valid"
		}
	}
	return "mismatch"
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("host", "", true, 253)
	service := q.String("service", "smtp", false, 5)
	port := q.Int("port", 0, 0, 65535)
	resolver := q.String("resolver", "system", false, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if service != "smtp" && service != "https" {
		httpapi.Error(w, r, httpapi.InvalidInput("service", "service must be smtp or https"))
		return
	}
	if port == 0 {
		port = map[string]int{"smtp": 25, "https": 443}[service]
	}
	name, err := target.NormalizeHostname(strings.TrimSpace(raw))
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("host", "enter a domain or host name"))
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, resolver, dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	res := Result{Input: name, Service: service, Port: port, Resolver: srv.Label, Hosts: []Host{}}
	names := []string{name}
	if service == "smtp" {
		if msg, _, err := dnsx.Query(ctx, srv, name, dns.TypeMX); err == nil {
			var mx []string
			for _, rr := range msg.Answer {
				if m, ok := rr.(*dns.MX); ok && m.Mx != "." && len(mx) < MaxHosts {
					mx = append(mx, strings.TrimSuffix(strings.ToLower(m.Mx), "."))
				}
			}
			if len(mx) > 0 {
				names, res.MX = mx, true
			}
		}
	}
	res.Hosts = make([]Host, len(names))
	var wg sync.WaitGroup
	for i, n := range names {
		wg.Go(func() { res.Hosts[i] = t.check(ctx, srv, n, service, port) })
	}
	wg.Wait()
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}

func (t *Tool) check(ctx context.Context, srv dnsx.Server, name, service string, port int) Host {
	h := Host{Name: name, TLSAName: "_" + strconv.Itoa(port) + "._tcp." + name, Records: []TLSA{}}
	m := dnsx.NewQuery(h.TLSAName, dns.TypeTLSA, true)
	if msg, _, err := dnsx.Exchange(ctx, srv, m); err != nil {
		h.Error = "TLSA lookup failed: " + httpapi.ToAPIError(dnslookup.Upstream(err)).Message
	} else {
		h.Rcode, h.DNSSEC = dnsx.RcodeName(msg.Rcode), msg.AuthenticatedData
		for _, rr := range msg.Answer {
			if tl, ok := rr.(*dns.TLSA); ok {
				h.Records = append(h.Records, TLSA{Usage: int(tl.Usage), Selector: int(tl.Selector), MatchingType: int(tl.MatchingType), Data: strings.ToLower(tl.Certificate), MatchedCert: -1})
			}
		}
	}
	tgt, err := t.deps.Target.Resolve(ctx, name, target.Opts{Field: "host"})
	if err != nil {
		h.Error = httpapi.ToAPIError(err).Message
		h.Verdict = Verdict(h)
		return h
	}
	dial := port
	if t.Port != 0 {
		dial = t.Port
	}
	addr := tgt.Resolved[0]
	for _, a := range tgt.Resolved {
		if a.Is4() {
			addr = a
			break
		}
	}
	h.Address = addr.String()
	opts := tlsx.Options{ServerName: name, Timeout: 15 * time.Second}
	if service == "smtp" {
		opts.StartTLS = "smtp"
	}
	s, err := tlsx.Handshake(ctx, netip.AddrPortFrom(addr, uint16(dial)).String(), opts) //nolint:gosec // port validated 1-65535
	if err != nil {
		h.Error = "TLS failed: " + err.Error()
	} else {
		h.TLS = &Summary{Version: s.Version, Cipher: s.Cipher, Chain: s.Chain, Verified: s.Verified, VerifyError: s.VerifyError}
		MatchChain(h.Records, s.Certs)
	}
	h.Verdict = Verdict(h)
	return h
}
