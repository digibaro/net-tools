// Package irr implements BGP-10: AS-SET / route-set expansion with the allow-listed bgpq4
// binary against the default IRR mirror (rr.ntt.net). The IRR host is not user-selectable.
package irr

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/execx"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Timeout bounds one bgpq4 run; output is capped by execx.MaxOutput (1 MiB).
const Timeout = 30 * time.Second

// Sources are the IRR databases a user may restrict the lookup to.
var Sources = []string{"RPKI", "RIPE", "ARIN", "APNIC", "AFRINIC", "LACNIC", "RADB", "NTTCOM", "LEVEL3", "ALTDB", "IDNIC", "JPIRR"}

// setRe matches AS numbers and (hierarchical) as-set / route-set names.
var setRe = regexp.MustCompile(`(?i)^(AS\d{1,10}|((AS\d{1,10}|AS-[A-Z0-9_-]+|RS-[A-Z0-9_-]+):)*(AS|RS)-[A-Z0-9_-]+)$`)

// ValidSet reports whether s is an ASN or as-set/route-set name.
func ValidSet(s string) bool { return len(s) <= 128 && setRe.MatchString(s) }

// Options for one expansion.
type Options struct {
	Set       string
	Mode      string // prefixes | asns
	Family    int    // 4 | 6
	Aggregate bool
	Sources   []string
}

// Args builds the bgpq4 argv.
func Args(o Options) ([]string, error) {
	set, err := execx.UserValue(strings.ToUpper(o.Set))
	if err != nil {
		return nil, err //nolint:wrapcheck // validation
	}
	args := []string{"-j"}
	if len(o.Sources) > 0 {
		args = append(args, "-S", strings.Join(o.Sources, ","))
	}
	if o.Mode == "asns" {
		return append(args, "-l", "asns", "-t", set), nil
	}
	args = append(args, "-l", "prefixes", "-"+map[int]string{4: "4", 6: "6"}[o.Family])
	if o.Aggregate {
		args = append(args, "-A")
	}
	return append(args, set), nil
}

// Prefix is one prefix-list entry.
type Prefix struct {
	Prefix       string `json:"prefix"`
	Exact        bool   `json:"exact"`
	GreaterEqual int    `json:"greaterEqual,omitempty"`
	LessEqual    int    `json:"lessEqual,omitempty"`
}

// Result is the response payload.
type Result struct {
	Set       string   `json:"set"`
	Mode      string   `json:"mode"`
	Family    int      `json:"family,omitempty"`
	Aggregate bool     `json:"aggregate"`
	Sources   []string `json:"sources"`
	Prefixes  []Prefix `json:"prefixes"`
	ASNs      []int64  `json:"asns"`
}

// Parse decodes bgpq4 -j output.
func Parse(out []byte, mode string) (prefixes []Prefix, asns []int64, err error) {
	if mode == "asns" {
		var d struct {
			ASNs []int64 `json:"asns"`
		}
		if err := json.Unmarshal(out, &d); err != nil {
			return nil, nil, err //nolint:wrapcheck // classified by caller
		}
		return []Prefix{}, d.ASNs, nil
	}
	var d struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
			Exact  bool   `json:"exact"`
			GE     int    `json:"greater-equal"`
			LE     int    `json:"less-equal"`
		} `json:"prefixes"`
	}
	if err := json.Unmarshal(out, &d); err != nil {
		return nil, nil, err //nolint:wrapcheck // classified by caller
	}
	prefixes = make([]Prefix, 0, len(d.Prefixes))
	for _, p := range d.Prefixes {
		prefixes = append(prefixes, Prefix{Prefix: p.Prefix, Exact: p.Exact, GreaterEqual: p.GE, LessEqual: p.LE})
	}
	return prefixes, []int64{}, nil
}

// Tool is bgp.irr.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string { return "bgp.irr" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	if d.Exec == nil || !d.Exec.Available("bgpq4") {
		return false, tools.ReasonBinaryMissing
	}
	return true, ""
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(Timeout+5*time.Second)).Get("/bgp/irr", t.handle)
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	o := Options{
		Set:       strings.TrimSpace(q.String("set", "", true, 128)),
		Mode:      q.Enum("mode", "prefixes", "prefixes", "asns"),
		Aggregate: q.Bool("aggregate", true),
	}
	if q.Enum("family", "4", "4", "6") == "6" {
		o.Family = 6
	} else {
		o.Family = 4
	}
	srcRaw := q.String("sources", "", false, 200)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if !ValidSet(o.Set) {
		httpapi.Error(w, r, httpapi.InvalidInput("set", "enter an AS number or an as-set / route-set name, such as AS-EXAMPLE or AS65000:AS-CUSTOMERS"))
		return
	}
	o.Set = strings.ToUpper(o.Set)
	o.Sources = []string{}
	for _, s := range strings.Split(srcRaw, ",") {
		s = strings.ToUpper(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		if !slices.Contains(Sources, s) {
			httpapi.Error(w, r, httpapi.InvalidInput("sources", "unknown IRR source "+s))
			return
		}
		if !slices.Contains(o.Sources, s) {
			o.Sources = append(o.Sources, s)
		}
	}
	args, err := Args(o)
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("set", "invalid set name"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), Timeout)
	defer cancel()
	var out strings.Builder
	stderr, err := t.deps.Exec.Run(ctx, "bgpq4", args, func(line string) {
		out.WriteString(line)
		out.WriteByte('\n')
	})
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeTimeout, Message: "the IRR lookup took longer than 30 s; try a smaller set or fewer sources"})
		return
	case err != nil:
		msg := strings.TrimSpace(stderr)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "bgpq4 failed: " + msg, Err: err})
		return
	}
	prefixes, asns, err := Parse([]byte(out.String()), o.Mode)
	if err != nil {
		if out.Len() >= execx.MaxOutput-1024 {
			httpapi.Error(w, r, httpapi.InvalidInput("set", "the expansion is larger than 1 MiB; enable aggregation or expand a smaller set"))
			return
		}
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "unreadable bgpq4 output", Err: err})
		return
	}
	res := Result{Set: o.Set, Mode: o.Mode, Aggregate: o.Aggregate && o.Mode == "prefixes", Sources: o.Sources, Prefixes: prefixes, ASNs: asns}
	if o.Mode == "prefixes" {
		res.Family = o.Family
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
