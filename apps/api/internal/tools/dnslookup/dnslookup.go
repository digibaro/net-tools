// Package dnslookup implements DNS-01: query any record type against a chosen resolver.
package dnslookup

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// RecordTypes offered by the tool (docs/API_SPEC.md).
var RecordTypes = []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA", "SRV", "CAA", "PTR", "DS", "DNSKEY", "HTTPS", "SVCB", "TLSA", "NAPTR", "ANY"}

// Tool is DNS lookup.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.lookup" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(15*time.Second)).Get("/dns/lookup", t.handle)
}

// Flags are the header bits of the answer.
type Flags struct {
	Authoritative      bool `json:"aa"`
	Truncated          bool `json:"tc"`
	RecursionDesired   bool `json:"rd"`
	RecursionAvailable bool `json:"ra"`
	AuthenticatedData  bool `json:"ad"`
	CheckingDisabled   bool `json:"cd"`
}

// Result is the response payload.
type Result struct {
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Resolver   string        `json:"resolver"`
	Transport  string        `json:"transport"`
	Rcode      string        `json:"rcode"`
	Flags      Flags         `json:"flags"`
	Answer     []dnsx.Record `json:"answer"`
	Authority  []dnsx.Record `json:"authority"`
	Additional []dnsx.Record `json:"additional"`
	RTTMs      float64       `json:"rttMs"`
}

// Ms converts a duration to milliseconds with 3 decimals.
func Ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// ParseName validates a query name (IDNA-normalized; IP addresses become their PTR name).
func ParseName(raw string, qtype uint16) (string, error) {
	s := strings.TrimSpace(raw)
	if r, err := dns.ReverseAddr(s); err == nil && qtype == dns.TypePTR {
		return strings.TrimSuffix(r, "."), nil
	}
	n, err := target.NormalizeHostname(s)
	if err != nil {
		return "", httpapi.InvalidInput("name", "not a valid domain name")
	}
	return n, nil
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	rawName := q.String("name", "", true, 253)
	typ := q.Enum("type", "A", RecordTypes...)
	resolver := q.String("resolver", "system", false, 300)
	transport := q.Enum("transport", "udp", "udp", "tcp", "dot", "doh")
	dnssec := q.Bool("dnssec", false)
	family := q.Family()
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	qtype, _ := dnsx.ParseType(typ)
	name, err := ParseName(rawName, qtype)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, resolver, dnsx.Transport(transport), family)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	start := time.Now()
	msg, rtt, err := dnsx.Exchange(r.Context(), srv, dnsx.NewQuery(name, qtype, dnssec))
	if err != nil {
		httpapi.Error(w, r, upstream(err))
		return
	}
	res := Result{
		Name: name, Type: typ, Resolver: srv.Label, Transport: transport,
		Rcode: dnsx.RcodeName(msg.Rcode),
		Flags: Flags{
			Authoritative: msg.Authoritative, Truncated: msg.Truncated, RecursionDesired: msg.RecursionDesired,
			RecursionAvailable: msg.RecursionAvailable, AuthenticatedData: msg.AuthenticatedData, CheckingDisabled: msg.CheckingDisabled,
		},
		Answer: dnsx.Records(msg.Answer), Authority: dnsx.Records(msg.Ns), Additional: dnsx.Records(msg.Extra),
		RTTMs: Ms(rtt),
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{
		Tool: t.ID(), Vantage: "server", DurationMs: Ms(time.Since(start)),
		Target: &httpapi.TargetMeta{Input: rawName, Used: srv.Addr.String()},
	})
}

// upstream maps query failures: deadline → TIMEOUT, others → PROVIDER_ERROR with a safe message.
func upstream(err error) error {
	a := httpapi.ToAPIError(err)
	if a.Code == httpapi.CodeTimeout {
		return a
	}
	return &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "the DNS server did not answer: " + firstLine(err.Error()), Err: err}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
