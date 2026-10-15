// Package serverinfo implements CON-13: interfaces, routes and resolvers of the api container
// (or host, in LAN mode). Disabled by default in the public profile.
package serverinfo

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
)

// Interface is one network interface.
type Interface struct {
	Name      string   `json:"name"`
	MTU       int      `json:"mtu"`
	MAC       string   `json:"mac"`
	Flags     []string `json:"flags"`
	Addresses []string `json:"addresses"`
}

// Result is the response payload.
type Result struct {
	Hostname   string      `json:"hostname"`
	LANMode    bool        `json:"lanMode"`
	Interfaces []Interface `json:"interfaces"`
	Routes     []Route     `json:"routes"`
	Gateways   []string    `json:"gateways"`
	Resolvers  Resolv      `json:"resolvers"`
}

// Tool is server network info.
type Tool struct {
	// Proc/ResolvConf paths are overridable for tests.
	RouteV4, RouteV6, ResolvConf string
	deps                         tools.Deps
}

func (*Tool) ID() string { return "net.serverinfo" }

func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) {
	if !d.Config.Features.ServerInfo {
		return false, tools.ReasonDisabledByAdmin
	}
	return true, ""
}

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(5*time.Second)).Get("/server/netinfo", t.handle)
}

func read(path, def string) string {
	if path == "" {
		path = def
	}
	b, err := os.ReadFile(path) //nolint:gosec // fixed /proc and /etc paths
	if err != nil {
		return ""
	}
	return string(b)
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	res := Result{Hostname: host, LANMode: t.deps.Config.Features.LAN, Interfaces: []Interface{}, Gateways: []string{}}
	ifaces, err := net.Interfaces()
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	for _, ifc := range ifaces {
		i := Interface{Name: ifc.Name, MTU: ifc.MTU, MAC: ifc.HardwareAddr.String(), Flags: strings.Split(ifc.Flags.String(), "|"), Addresses: []string{}}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			i.Addresses = append(i.Addresses, a.String())
		}
		res.Interfaces = append(res.Interfaces, i)
	}
	res.Routes = append(ParseRouteV4(read(t.RouteV4, "/proc/net/route")), ParseRouteV6(read(t.RouteV6, "/proc/net/ipv6_route"))...)
	if res.Routes == nil {
		res.Routes = []Route{}
	}
	for _, rt := range res.Routes {
		if rt.Default && rt.Gateway != "" {
			res.Gateways = append(res.Gateways, rt.Gateway+" ("+rt.Interface+")")
		}
	}
	res.Resolvers = ParseResolvConf(read(t.ResolvConf, "/etc/resolv.conf"))
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
