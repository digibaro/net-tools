// Package dnswildcard implements DNS-11: detect wildcard records by querying random labels
// that cannot exist unless a wildcard (*.domain) answers for them.
package dnswildcard

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Probes per check and the record types asked for.
const Probes = 3

// Types are checked for each random name.
var Types = []uint16{dns.TypeA, dns.TypeAAAA, dns.TypeCNAME, dns.TypeMX, dns.TypeTXT}

// Tool is dns.wildcard.
type Tool struct {
	deps tools.Deps
	// Label overrides the random label generator (tests).
	Label func() string
}

func (*Tool) ID() string                                { return "dns.wildcard" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Label == nil {
		t.Label = randomLabel
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(20*time.Second)).Get("/dns/wildcard", t.handle)
}

func randomLabel() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "nt-" + hex.EncodeToString(b)
}

// Probe is one random name and what it returned.
type Probe struct {
	Name    string        `json:"name"`
	Rcode   string        `json:"rcode"`
	Answers []dnsx.Record `json:"answers"`
}

// Result is the DNS-11 payload.
type Result struct {
	Domain   string   `json:"domain"`
	Wildcard bool     `json:"wildcard"`
	Types    []string `json:"types"`
	// Consistent: every probe got the same answers (a plain wildcard); differing answers point to
	// load balancing or per-name synthesis.
	Consistent bool    `json:"consistent"`
	Probes     []Probe `json:"probes"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("domain", "", true, 230)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	domain, err := target.NormalizeHostname(strings.TrimPrefix(strings.TrimSpace(raw), "*."))
	if err != nil || !strings.Contains(domain, ".") {
		httpapi.Error(w, r, httpapi.InvalidInput("domain", "enter a domain name such as example.com"))
		return
	}
	sys, err := dnsx.SystemServer(dnsx.UDP)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	res := Result{Domain: domain, Types: []string{}, Probes: []Probe{}}
	types := map[string]bool{}
	var signatures []string
	for range Probes {
		name := t.Label() + "." + domain
		p := Probe{Name: name, Rcode: "NXDOMAIN", Answers: []dnsx.Record{}}
		var sig []string
		for _, qt := range Types {
			m, _, err := dnsx.Query(r.Context(), sys, name, qt)
			if err != nil {
				httpapi.Error(w, r, dnslookup.Upstream(err))
				return
			}
			if m.Rcode == dns.RcodeSuccess {
				p.Rcode = "NOERROR"
			}
			for _, rec := range dnsx.Records(m.Answer) {
				p.Answers = append(p.Answers, rec)
				types[rec.Type] = true
				sig = append(sig, rec.Type+" "+rec.Data)
			}
		}
		slices.Sort(sig)
		signatures = append(signatures, strings.Join(sig, "|"))
		res.Wildcard = res.Wildcard || len(p.Answers) > 0
		res.Probes = append(res.Probes, p)
	}
	for ty := range types {
		res.Types = append(res.Types, ty)
	}
	slices.Sort(res.Types)
	res.Consistent = res.Wildcard && !slices.ContainsFunc(signatures, func(s string) bool { return s != signatures[0] })
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}
