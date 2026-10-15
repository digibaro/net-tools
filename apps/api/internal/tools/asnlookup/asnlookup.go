// Package asnlookup implements BGP-01: origin ASN, prefix, AS name and RIR for an IP.
package asnlookup

import (
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/asn"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// Tool is IP → ASN.
type Tool struct {
	deps tools.Deps
	TXT  asn.TXTFunc
}

func (*Tool) ID() string                                { return "bgp.ip2asn" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.TXT == nil {
		t.TXT = asn.SystemTXT
	}
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(10*time.Second)).Get("/asn/lookup", t.handle)
}

// Result is the response payload.
type Result struct {
	IP        string       `json:"ip"`
	Announced bool         `json:"announced"`
	Origins   []asn.Origin `json:"origins"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("ip", "", true, 64)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ip, err := netip.ParseAddr(strings.Trim(raw, "[]"))
	if err != nil || ip.Zone() != "" {
		httpapi.Error(w, r, httpapi.InvalidInput("ip", "enter an IPv4 or IPv6 address"))
		return
	}
	origins, err := asn.Lookup(r.Context(), t.TXT, ip)
	if err != nil {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	if origins == nil {
		origins = []asn.Origin{}
	}
	httpapi.JSON(w, r, Result{IP: ip.Unmap().String(), Announced: len(origins) > 0, Origins: origins}, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
