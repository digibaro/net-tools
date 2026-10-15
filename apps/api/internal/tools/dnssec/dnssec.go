package dnssec

import (
	"net/http"
	"net/netip"
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

// DefaultServer is the recursive resolver used to fetch records (DO+CD; we validate ourselves).
var DefaultServer = dnsx.Server{Label: "Cloudflare (1.1.1.1)", Addr: netip.MustParseAddrPort("1.1.1.1:53"), Transport: dnsx.TCP}

// Tool is dns.dnssec.
type Tool struct {
	deps tools.Deps
	// Server and Anchors override the resolver and root trust anchors (tests).
	Server  *dnsx.Server
	Anchors []*dns.DS
	Now     func() time.Time
}

func (*Tool) ID() string                                { return "dns.dnssec" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Server == nil {
		s := DefaultServer
		t.Server = &s
	}
	if t.Anchors == nil {
		t.Anchors = RootAnchors
	}
	if t.Now == nil {
		t.Now = time.Now
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(25*time.Second)).Get("/dns/dnssec", t.handle)
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("name", "", true, 253)
	typ := q.Enum("type", "A", "A", "AAAA", "MX", "TXT", "NS", "SOA", "CNAME", "CAA", "DNSKEY", "DS", "HTTPS", "TLSA", "SRV")
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	name, err := target.NormalizeHostname(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("name", "not a valid domain name"))
		return
	}
	qt, _ := dnsx.ParseType(typ)
	v := &Validator{Server: *t.Server, Anchors: t.Anchors, Now: t.Now}
	res, err := v.Validate(r.Context(), name, qt)
	if err != nil {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw, Used: t.Server.Label}})
}
