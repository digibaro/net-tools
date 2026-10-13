package serverinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

const routeV4 = `Iface	Destination	Gateway 	Flags	RefCnt	Use	Metric	Mask		MTU	Window	IRTT
eth0	00000000	010013AC	0003	0	0	0	00000000	0	0	0
eth0	000013AC	00000000	0001	0	0	0	0000FFFF	0	0	0
`

const routeV6 = `00000000000000000000000000000000 00 00000000000000000000000000000000 00 fe800000000000000000000000000001 00000400 00000001 00000000 00000003 eth0
20010db8000000000000000000000000 30 00000000000000000000000000000000 00 00000000000000000000000000000000 00000100 00000001 00000000 00000001 eth0
00000000000000000000000000000001 80 00000000000000000000000000000000 00 00000000000000000000000000000000 00000000 00000002 00000000 80200001 lo
`

func TestParsers(t *testing.T) {
	v4 := ParseRouteV4(routeV4)
	if len(v4) != 2 || v4[0].Destination != "0.0.0.0/0" || v4[0].Gateway != "172.19.0.1" || !v4[0].Default || v4[1].Destination != "172.19.0.0/16" || v4[1].Gateway != "" {
		t.Errorf("v4 = %+v", v4)
	}
	v6 := ParseRouteV6(routeV6)
	if len(v6) != 2 || v6[0].Gateway != "fe80::1" || !v6[0].Default || v6[0].Metric != 1024 || v6[1].Destination != "2001:db8::/48" {
		t.Errorf("v6 = %+v", v6)
	}
	r := ParseResolvConf("# comment\nnameserver 127.0.0.11\nsearch example.lan corp.example\noptions ndots:0\n")
	if strings.Join(r.Nameservers, ",") != "127.0.0.11" || len(r.Search) != 2 || r.Options[0] != "ndots:0" {
		t.Errorf("resolv = %+v", r)
	}
}

func TestDisabledInPublicProfile(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{})
	if r := env.Get("/api/v1/server/netinfo"); r.Status != 404 {
		t.Errorf("must not be mounted in public profile: %d", r.Status)
	}
	if !strings.Contains(env.Get("/api/v1/capabilities").Raw.Body.String(), `"net.serverinfo":{"enabled":false,"reason":"disabled_by_admin"}`) {
		t.Error("capability must report disabled_by_admin")
	}
}

func TestEnabled(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		_ = os.WriteFile(p, []byte(body), 0o600)
		return p
	}
	tool := &Tool{RouteV4: write("r4", routeV4), RouteV6: write("r6", routeV6), ResolvConf: write("rc", "nameserver 9.9.9.9\n")}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_PROFILE": "homelab"}})
	var res Result
	env.Get("/api/v1/server/netinfo").Must(t).Decode(t, &res)
	if len(res.Interfaces) == 0 || len(res.Routes) != 4 || res.Gateways[0] != "172.19.0.1 (eth0)" || res.Resolvers.Nameservers[0] != "9.9.9.9" {
		t.Errorf("res = %+v", res)
	}
}
