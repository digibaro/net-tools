// Package portcheck implements CON-05: is a TCP (or, best effort, UDP) port open from the server.
package portcheck

import (
	"context"
	"errors"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools/apps/api/internal/tools/rttstats"
)

// MaxPorts per request (docs/API_SPEC.md: ports ≤ 10).
const MaxPorts = 10

// Services names common ports for display.
var Services = map[int]string{
	21: "ftp", 22: "ssh", 23: "telnet", 25: "smtp", 53: "dns", 80: "http", 110: "pop3", 123: "ntp", 143: "imap",
	161: "snmp", 389: "ldap", 443: "https", 445: "smb", 465: "smtps", 500: "isakmp", 587: "submission", 636: "ldaps",
	853: "dns-over-tls", 993: "imaps", 995: "pop3s", 1194: "openvpn", 1433: "mssql", 1521: "oracle", 3306: "mysql",
	3389: "rdp", 4500: "ipsec-nat-t", 5060: "sip", 5432: "postgresql", 5900: "vnc", 6379: "redis", 8080: "http-alt",
	8443: "https-alt", 9200: "elasticsearch", 27017: "mongodb", 51820: "wireguard",
}

// Tool is the port check.
type Tool struct{ deps tools.Deps }

func (*Tool) ID() string                                { return "net.portcheck" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(15*time.Second)).Get("/port/check", t.handle)
}

// Row is one port's result.
type Row struct {
	Port    int     `json:"port"`
	Proto   string  `json:"proto"`
	State   string  `json:"state"`
	Service string  `json:"service"`
	RTTMs   float64 `json:"rttMs"`
	Detail  string  `json:"detail"`
}

// Result is the response payload.
type Result struct {
	Target  string `json:"target"`
	Address string `json:"address"`
	Rows    []Row  `json:"rows"`
}

// ParsePorts accepts "22,80,443" and small ranges "8000-8003", at most MaxPorts in total.
func ParsePorts(s string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lo, hi, isRange := strings.Cut(part, "-")
		a, err1 := strconv.Atoi(strings.TrimSpace(lo))
		b := a
		var err2 error
		if isRange {
			b, err2 = strconv.Atoi(strings.TrimSpace(hi))
		}
		if err1 != nil || err2 != nil || a < 1 || b > 65535 || b < a {
			return nil, errors.New("ports must be numbers or ranges from 1 to 65535")
		}
		for p := a; p <= b; p++ {
			if !slices.Contains(out, p) {
				out = append(out, p)
			}
			if len(out) > MaxPorts {
				return nil, errors.New("at most 10 ports per check")
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("at least one port is required")
	}
	return out, nil
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("target", "", true, 300)
	portsRaw := q.String("ports", "", true, 200)
	proto := q.Enum("proto", "tcp", "tcp", "udp")
	family := q.Family()
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ports, err := ParsePorts(portsRaw)
	if err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("ports", err.Error()))
		return
	}
	tgt, err := t.deps.Target.Resolve(r.Context(), raw, target.Opts{Family: family})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	res := Result{Target: raw, Address: tgt.IP.String(), Rows: make([]Row, len(ports))}
	var wg sync.WaitGroup
	for i, p := range ports {
		wg.Go(func() {
			if proto == "udp" {
				res.Rows[i] = probeUDP(ctx, tgt.AddrPort(p), p)
			} else {
				res.Rows[i] = probeTCP(ctx, tgt.AddrPort(p), p)
			}
		})
	}
	wg.Wait()
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw, Resolved: addrs(tgt), Used: tgt.IP.String()}})
}

func addrs(t target.Target) []string {
	out := make([]string, len(t.Resolved))
	for i, a := range t.Resolved {
		out[i] = a.String()
	}
	return out
}

func probeTCP(ctx context.Context, addr string, port int) Row {
	row := Row{Port: port, Proto: "tcp", Service: Services[port]}
	d := net.Dialer{Timeout: 4 * time.Second}
	start := time.Now()
	c, err := d.DialContext(ctx, "tcp", addr)
	switch {
	case err == nil:
		row.State, row.RTTMs = "open", rttstats.Ms(time.Since(start))
		_ = c.Close()
	case errors.Is(err, syscall.ECONNREFUSED):
		row.State, row.Detail = "closed", "connection refused"
	default:
		row.State, row.Detail = "filtered", "no response (timeout or dropped)"
	}
	return row
}

// probeUDP sends an empty datagram. An ICMP port-unreachable shows up as ECONNREFUSED on the
// connected socket (closed); silence means open or filtered; any reply means open.
func probeUDP(ctx context.Context, addr string, port int) Row {
	row := Row{Port: port, Proto: "udp", Service: Services[port]}
	d := net.Dialer{Timeout: 3 * time.Second}
	c, err := d.DialContext(ctx, "udp", addr)
	if err != nil {
		row.State, row.Detail = "error", "could not send"
		return row
	}
	defer c.Close()
	start := time.Now()
	_ = c.SetDeadline(time.Now().Add(2500 * time.Millisecond))
	buf := make([]byte, 512)
	for range 2 {
		if _, err := c.Write([]byte{}); err != nil {
			break
		}
		_, err = c.Read(buf)
		if err == nil {
			row.State, row.RTTMs = "open", rttstats.Ms(time.Since(start))
			return row
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			row.State, row.Detail = "closed", "ICMP port unreachable"
			return row
		}
	}
	row.State, row.Detail = "open|filtered", "no reply (UDP services often stay silent)"
	return row
}
