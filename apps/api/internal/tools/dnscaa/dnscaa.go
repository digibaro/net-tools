// Package dnscaa implements DNS-10: CAA lookup with tree climbing (RFC 8659 §3).
package dnscaa

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
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Tool is the CAA checker.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "dns.caa" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(20*time.Second)).Get("/dns/caa", t.handle)
}

// Record is a parsed CAA record.
type Record struct {
	Flag     uint8  `json:"flag"`
	Critical bool   `json:"critical"`
	Tag      string `json:"tag"`
	Value    string `json:"value"`
}

// Step is one name examined while climbing.
type Step struct {
	Name    string   `json:"name"`
	Rcode   string   `json:"rcode"`
	Records []Record `json:"records"`
}

// Result is the response payload.
type Result struct {
	Name string `json:"name"`
	// RelevantName is where the effective CAA set was found ("" = none: any CA may issue).
	RelevantName string   `json:"relevantName"`
	Steps        []Step   `json:"steps"`
	Issue        []string `json:"issue"`
	IssueWild    []string `json:"issueWild"`
	Iodef        []string `json:"iodef"`
	// UnknownCritical lists critical records with unknown tags: CAs must refuse to issue.
	UnknownCritical []string `json:"unknownCritical"`
	Resolver        string   `json:"resolver"`
}

var knownTags = map[string]bool{"issue": true, "issuewild": true, "iodef": true, "issuemail": true, "issuevmc": true, "contactemail": true, "contactphone": true}

// Climb returns name and its parents, excluding the root: a.b.example.com, b.example.com, …, com.
func Climb(name string) []string {
	labels := strings.Split(name, ".")
	out := make([]string, 0, len(labels))
	for i := range labels {
		out = append(out, strings.Join(labels[i:], "."))
	}
	return out
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("name", "", true, 253)
	resolver := q.String("resolver", "system", false, 300)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	name, err := target.NormalizeHostname(strings.TrimPrefix(strings.TrimSpace(raw), "*."))
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("name", "not a valid domain name"))
		return
	}
	srv, err := dnsx.ResolveServer(r.Context(), t.deps.Target, resolver, dnsx.UDP, 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	start := time.Now()
	res := Result{Name: name, Steps: []Step{}, Issue: []string{}, IssueWild: []string{}, Iodef: []string{}, UnknownCritical: []string{}, Resolver: srv.Label}
	for _, n := range Climb(name) {
		msg, _, err := dnsx.Query(r.Context(), srv, n, dns.TypeCAA)
		if err != nil {
			httpapi.Error(w, r, dnslookup.Upstream(err))
			return
		}
		step := Step{Name: n, Rcode: dnsx.RcodeName(msg.Rcode), Records: []Record{}}
		for _, rr := range msg.Answer {
			if c, ok := rr.(*dns.CAA); ok {
				step.Records = append(step.Records, Record{Flag: c.Flag, Critical: c.Flag&128 != 0, Tag: strings.ToLower(c.Tag), Value: c.Value})
			}
		}
		res.Steps = append(res.Steps, step)
		if msg.Rcode == dns.RcodeServerFailure {
			break
		}
		if len(step.Records) > 0 {
			res.RelevantName = n
			for _, rec := range step.Records {
				switch rec.Tag {
				case "issue":
					res.Issue = append(res.Issue, rec.Value)
				case "issuewild":
					res.IssueWild = append(res.IssueWild, rec.Value)
				case "iodef":
					res.Iodef = append(res.Iodef, rec.Value)
				}
				if rec.Critical && !knownTags[rec.Tag] {
					res.UnknownCritical = append(res.UnknownCritical, rec.Tag)
				}
			}
			break
		}
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", DurationMs: dnslookup.Ms(time.Since(start)), Target: &httpapi.TargetMeta{Input: raw, Used: srv.Addr.String()}})
}
