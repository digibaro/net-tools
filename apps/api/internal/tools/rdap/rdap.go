// Package rdap implements BGP-07 (RDAP lookup) and BGP-08 (domain expiry via RDAP).
package rdap

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	ordap "github.com/openrdap/rdap"
	"github.com/openrdap/rdap/bootstrap"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/netx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/whois"
)

// Service wraps a policy-checked RDAP client with a shared bootstrap cache.
type Service struct {
	once   sync.Once
	client *ordap.Client
	// BootstrapURL overrides https://data.iana.org/rdap/ (tests).
	BootstrapURL string
}

func (s *Service) init(res *target.Resolver, ua string) {
	s.once.Do(func() {
		hc := netx.PolicyClient(res, ua, 15*time.Second)
		bs := &bootstrap.Client{HTTP: hc}
		if s.BootstrapURL != "" {
			bs.BaseURL, _ = url.Parse(s.BootstrapURL)
		}
		s.client = &ordap.Client{HTTP: hc, Bootstrap: bs, UserAgent: ua}
	})
}

// Entity is a contact (registrar, registrant, abuse…).
type Entity struct {
	Handle  string   `json:"handle"`
	Roles   []string `json:"roles"`
	Name    string   `json:"name"`
	Org     string   `json:"org"`
	Email   string   `json:"email"`
	Country string   `json:"country"`
}

// Event is a dated lifecycle event.
type Event struct {
	Action string `json:"action"`
	Date   string `json:"date"`
}

// Record is a normalized RDAP object.
type Record struct {
	Query       string   `json:"query"`
	Kind        string   `json:"kind"`
	Handle      string   `json:"handle"`
	Name        string   `json:"name"`
	Status      []string `json:"status"`
	Events      []Event  `json:"events"`
	Nameservers []string `json:"nameservers"`
	Entities    []Entity `json:"entities"`
	Registrar   string   `json:"registrar"`
	Country     string   `json:"country"`
	Range       string   `json:"range"`
	Type        string   `json:"type"`
	Port43      string   `json:"port43"`
	Server      string   `json:"server"`
}

func entities(list []ordap.Entity) []Entity {
	out := []Entity{}
	for _, e := range list {
		x := Entity{Handle: e.Handle, Roles: e.Roles}
		if x.Roles == nil {
			x.Roles = []string{}
		}
		if e.VCard != nil {
			x.Name, x.Org, x.Email, x.Country = e.VCard.Name(), e.VCard.Org(), e.VCard.Email(), e.VCard.Country()
		}
		out = append(out, x)
		out = append(out, entities(e.Entities)...)
	}
	return out
}

func events(list []ordap.Event) []Event {
	out := []Event{}
	for _, e := range list {
		out = append(out, Event{Action: e.Action, Date: e.Date})
	}
	return out
}

func registrar(es []Entity) string {
	for _, e := range es {
		for _, r := range e.Roles {
			if r == "registrar" {
				return firstNonEmpty(e.Name, e.Org, e.Handle)
			}
		}
	}
	return ""
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Lookup queries RDAP for a domain, IP or ASN (kind detected like WHOIS).
func (s *Service) Lookup(ctx context.Context, res *target.Resolver, ua, raw string) (*Record, error) {
	s.init(res, ua)
	q, kind, err := whois.Normalize(raw)
	if err != nil {
		return nil, err //nolint:wrapcheck // client-facing
	}
	var req *ordap.Request
	switch kind {
	case "domain":
		req = ordap.NewDomainRequest(q)
	case "ip":
		req = ordap.NewIPRequest(net.IP(netip.MustParseAddr(q).AsSlice()))
	default:
		n, _ := strconv.ParseUint(strings.TrimPrefix(q, "AS"), 10, 32)
		req = ordap.NewAutnumRequest(uint32(n))
	}
	resp, err := s.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, mapErr(err)
	}
	rec := &Record{Query: q, Kind: kind, Status: []string{}, Events: []Event{}, Nameservers: []string{}, Entities: []Entity{}}
	if n := len(resp.HTTP); n > 0 && resp.HTTP[n-1].URL != "" {
		rec.Server = resp.HTTP[n-1].URL
	}
	switch o := resp.Object.(type) {
	case *ordap.Domain:
		rec.Handle, rec.Name, rec.Status, rec.Events, rec.Port43 = o.Handle, strings.ToLower(o.LDHName), nonNil(o.Status), events(o.Events), o.Port43
		for _, ns := range o.Nameservers {
			rec.Nameservers = append(rec.Nameservers, strings.ToLower(ns.LDHName))
		}
		rec.Entities = entities(o.Entities)
	case *ordap.IPNetwork:
		rec.Handle, rec.Name, rec.Status, rec.Events, rec.Port43 = o.Handle, o.Name, nonNil(o.Status), events(o.Events), o.Port43
		rec.Range, rec.Country, rec.Type = o.StartAddress+" – "+o.EndAddress, o.Country, o.Type
		rec.Entities = entities(o.Entities)
	case *ordap.Autnum:
		rec.Handle, rec.Name, rec.Status, rec.Events, rec.Port43 = o.Handle, o.Name, nonNil(o.Status), events(o.Events), o.Port43
		rec.Country, rec.Type = o.Country, o.Type
		if o.StartAutnum != nil && o.EndAutnum != nil {
			rec.Range = "AS" + strconv.FormatUint(uint64(*o.StartAutnum), 10) + " – AS" + strconv.FormatUint(uint64(*o.EndAutnum), 10)
		}
		rec.Entities = entities(o.Entities)
	default:
		return nil, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "unexpected RDAP response"}
	}
	rec.Registrar = registrar(rec.Entities)
	return rec, nil
}

func mapErr(err error) error {
	var ce *ordap.ClientError
	if errors.As(err, &ce) {
		switch ce.Type {
		case ordap.ObjectDoesNotExist:
			return &httpapi.APIError{Code: httpapi.CodeInvalidInput, Message: "no RDAP record exists for this query", Details: map[string]any{"field": "query"}, Err: err}
		case ordap.BootstrapNoMatch:
			return &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "no RDAP server is registered for this query (the TLD or registry may not offer RDAP)", Err: err}
		}
	}
	var coder httpapi.Coder
	if errors.As(err, &coder) {
		return coder
	}
	return &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "the RDAP server did not answer", Err: err}
}

// LookupTool is BGP-07.
type LookupTool struct {
	deps tools.Deps
	Svc  *Service
}

func (*LookupTool) ID() string                                { return "reg.rdap" }
func (*LookupTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *LookupTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Svc == nil {
		t.Svc = &Service{}
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(30*time.Second)).Get("/rdap", t.handle)
}

func (t *LookupTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("query", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	rec, err := t.Svc.Lookup(r.Context(), t.deps.Target, t.deps.UserAgent, raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	httpapi.JSON(w, r, rec, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw, Used: rec.Server}})
}

// ExpiryTool is BGP-08.
type ExpiryTool struct {
	deps tools.Deps
	Svc  *Service
	Now  func() time.Time
}

func (*ExpiryTool) ID() string                                { return "reg.expiry" }
func (*ExpiryTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *ExpiryTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Svc == nil {
		t.Svc = &Service{}
	}
	if t.Now == nil {
		t.Now = time.Now
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(30*time.Second)).Get("/domain/expiry", t.handle)
}

// Expiry is the BGP-08 payload.
type Expiry struct {
	Domain     string   `json:"domain"`
	Registered string   `json:"registered"`
	Expires    string   `json:"expires"`
	Updated    string   `json:"updated"`
	DaysLeft   *int     `json:"daysLeft"`
	Registrar  string   `json:"registrar"`
	Status     []string `json:"status"`
	Server     string   `json:"server"`
}

func (t *ExpiryTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("domain", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if _, kind, err := whois.Normalize(raw); err != nil || kind != "domain" {
		httpapi.Error(w, r, httpapi.InvalidInput("domain", "enter a domain name such as example.com"))
		return
	}
	rec, err := t.Svc.Lookup(r.Context(), t.deps.Target, t.deps.UserAgent, raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := Expiry{Domain: rec.Name, Registrar: rec.Registrar, Status: rec.Status, Server: rec.Server}
	for _, e := range rec.Events {
		switch e.Action {
		case "registration":
			out.Registered = e.Date
		case "expiration":
			out.Expires = e.Date
			if ts, err := time.Parse(time.RFC3339, e.Date); err == nil {
				d := int(ts.Sub(t.Now()).Hours() / 24)
				out.DaysLeft = &d
			}
		case "last changed":
			out.Updated = e.Date
		}
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw, Used: rec.Server}})
}
