// Package torcheck implements REP-04: is an IP a Tor exit (bulk exit list, refreshed hourly)
// and which relays use it (Onionoo).
package torcheck

import (
	"bufio"
	"bytes"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/providers"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/reputation"
)

// MaxRelays bounds the relay list.
const MaxRelays = 20

// Tool is rep.tor.
type Tool struct {
	deps tools.Deps
	// ListURL and OnionooBase override the Tor Project endpoints (tests).
	ListURL     string
	OnionooBase string
}

func (*Tool) ID() string                                { return "rep.tor" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.ListURL == "" {
		t.ListURL = "https://check.torproject.org/torbulkexitlist"
	}
	if t.OnionooBase == "" {
		t.OnionooBase = "https://onionoo.torproject.org"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(20*time.Second)).Get("/tor-exit", t.handle)
}

// Relay is a Tor relay using the address.
type Relay struct {
	Nickname    string   `json:"nickname"`
	Fingerprint string   `json:"fingerprint"`
	Flags       []string `json:"flags"`
	Running     bool     `json:"running"`
	FirstSeen   string   `json:"firstSeen"`
	LastSeen    string   `json:"lastSeen"`
	Country     string   `json:"country"`
	AS          string   `json:"as"`
	ASName      string   `json:"asName"`
	Exit        bool     `json:"exit"`
	URL         string   `json:"url"`
}

// Result is the REP-04 payload.
type Result struct {
	IP         string  `json:"ip"`
	ExitNode   bool    `json:"exitNode"`
	ListSize   int     `json:"listSize"`
	Relays     []Relay `json:"relays"`
	RelayError string  `json:"relayError"`
}

func (t *Tool) exitSet(r *http.Request) (map[netip.Addr]bool, error) {
	b, err := t.deps.Providers.Get(r.Context(), providers.Request{Provider: "Tor Project", URL: t.ListURL, TTL: time.Hour, Header: http.Header{"Accept": {"text/plain"}}})
	if err != nil {
		return nil, err //nolint:wrapcheck // client-safe
	}
	set := map[netip.Addr]bool{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		if a, err := netip.ParseAddr(strings.TrimSpace(sc.Text())); err == nil {
			set[a.Unmap()] = true
		}
	}
	return set, nil
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("ip", "", true, 64)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ip, err := reputation.PublicIP(raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	set, err := t.exitSet(r)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	out := Result{IP: ip.String(), ExitNode: set[ip], ListSize: len(set), Relays: []Relay{}}
	var details struct {
		Relays []struct {
			Nickname    string   `json:"nickname"`
			Fingerprint string   `json:"fingerprint"`
			Flags       []string `json:"flags"`
			Running     bool     `json:"running"`
			FirstSeen   string   `json:"first_seen"`
			LastSeen    string   `json:"last_seen"`
			Country     string   `json:"country"`
			AS          string   `json:"as"`
			ASName      string   `json:"as_name"`
		} `json:"relays"`
	}
	v := url.Values{"search": {ip.String()}, "fields": {"nickname,fingerprint,flags,running,first_seen,last_seen,country,as,as_name"}}
	// Relay details are a bonus: the exit verdict stands even if Onionoo is unavailable.
	if err := t.deps.Providers.GetJSON(r.Context(), providers.Request{Provider: "Onionoo", URL: t.OnionooBase + "/details?" + v.Encode(), TTL: time.Hour}, &details); err != nil {
		out.RelayError = httpapi.ToAPIError(err).Message
	}
	for i, rl := range details.Relays {
		if i >= MaxRelays {
			break
		}
		exit := false
		for _, f := range rl.Flags {
			exit = exit || f == "Exit"
		}
		if rl.Flags == nil {
			rl.Flags = []string{}
		}
		out.Relays = append(out.Relays, Relay{Nickname: rl.Nickname, Fingerprint: rl.Fingerprint, Flags: rl.Flags, Running: rl.Running, FirstSeen: rl.FirstSeen, LastSeen: rl.LastSeen, Country: strings.ToUpper(rl.Country), AS: rl.AS, ASName: rl.ASName, Exit: exit, URL: "https://metrics.torproject.org/rs.html#details/" + rl.Fingerprint})
	}
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "torproject"})
}
