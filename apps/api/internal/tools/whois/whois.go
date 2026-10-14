// Package whois implements BGP-06: WHOIS for domains, IPs and ASNs with referral following.
// It speaks port 43 itself so every server is resolved through the target policy and dialed
// by pinned IP (third-party WHOIS libraries dial hostnames directly).
package whois

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/limits"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// Limits.
const (
	MaxReferrals = 3
	MaxResponse  = 256 << 10
)

// Tool is WHOIS.
type Tool struct {
	deps tools.Deps
	// Root overrides whois.iana.org:43 (tests).
	Root string
	// anyPort lifts the WHOIS/RWHOIS port restriction (tests with ephemeral ports).
	anyPort bool
}

// whoisPorts are the only ports servers and referrals may use: WHOIS (43) and RWHOIS (4321).
func whoisPort(p int) bool { return p == 43 || p == 4321 }

func (*Tool) ID() string                                { return "reg.whois" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Root == "" {
		t.Root = "whois.iana.org:43"
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(45*time.Second)).Get("/whois", t.handle)
}

// Response is one server's answer.
type Response struct {
	Server    string `json:"server"`
	Query     string `json:"query"`
	Text      string `json:"text"`
	Truncated bool   `json:"truncated"`
	Error     string `json:"error"`
}

// Parsed holds commonly used fields from the most specific response.
type Parsed struct {
	Registrar   string   `json:"registrar"`
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Expires     string   `json:"expires"`
	Status      []string `json:"status"`
	Nameservers []string `json:"nameservers"`
	Org         string   `json:"org"`
	Country     string   `json:"country"`
	NetRange    string   `json:"netRange"`
}

// Result is the response payload.
type Result struct {
	Query  string     `json:"query"`
	Kind   string     `json:"kind"`
	Chain  []Response `json:"chain"`
	Parsed Parsed     `json:"parsed"`
}

var asnRe = regexp.MustCompile(`(?i)^(?:AS)?(\d{1,10})$`)

// Normalize classifies the query as domain, ip or asn and canonicalizes it.
func Normalize(raw string) (query, kind string, err error) {
	s := strings.TrimSpace(raw)
	if ip, e := netip.ParseAddr(strings.Trim(s, "[]")); e == nil && ip.Zone() == "" {
		return ip.String(), "ip", nil
	}
	if m := asnRe.FindStringSubmatch(s); m != nil {
		n, e := strconv.ParseUint(m[1], 10, 32)
		if e == nil {
			return "AS" + strconv.FormatUint(n, 10), "asn", nil
		}
	}
	d, e := target.NormalizeHostname(s)
	if e != nil || !strings.Contains(d, ".") {
		return "", "", httpapi.InvalidInput("query", "enter a domain, IP address or AS number")
	}
	return d, "domain", nil
}

var referralPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)^\s*refer:\s*(\S+)`),
	regexp.MustCompile(`(?im)^\s*whois:\s*(\S+)`),
	regexp.MustCompile(`(?im)^\s*ReferralServer:\s*(\S+)`),
	regexp.MustCompile(`(?im)^\s*Registrar WHOIS Server:\s*(\S+)`),
}

// Referral extracts the next server ("host" or "host:port") from a response, if any.
func Referral(text string) string {
	for _, re := range referralPatterns {
		if m := re.FindStringSubmatch(text); m != nil {
			s := strings.TrimSpace(m[1])
			s = strings.TrimPrefix(strings.TrimPrefix(s, "whois://"), "rwhois://")
			s = strings.TrimSuffix(strings.SplitN(s, "/", 2)[0], ".")
			if s != "" {
				return s
			}
		}
	}
	return ""
}

// queryFor formats the query for servers with special syntax.
func queryFor(server, q, kind string) string {
	if strings.HasPrefix(strings.ToLower(server), "whois.arin.net") {
		switch kind {
		case "ip":
			return "n + " + q
		case "asn":
			return "a + " + strings.TrimPrefix(q, "AS")
		}
	}
	return q
}

func (t *Tool) ask(ctx context.Context, server, query string) Response {
	resp := Response{Server: server, Query: query}
	hostport := server
	if _, _, err := net.SplitHostPort(server); err != nil {
		hostport = net.JoinHostPort(strings.Trim(server, "[]"), "43") // "[2001:db8::43]" has no port
	}
	tgt, err := t.deps.Target.Resolve(ctx, hostport, target.Opts{AllowPort: true, Field: "server"})
	if err != nil {
		resp.Error = httpapi.ToAPIError(err).Message
		return resp
	}
	port := tgt.Port
	if port == 0 {
		port = 43
	}
	if !t.anyPort && !whoisPort(port) {
		resp.Error = "referral to a non-WHOIS port refused"
		return resp
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	d := net.Dialer{}
	c, err := d.DialContext(ctx, "tcp", tgt.AddrPort(port))
	if err != nil {
		resp.Error = "connection failed"
		return resp
	}
	defer c.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetDeadline(dl)
	}
	if _, err := fmt.Fprintf(c, "%s\r\n", query); err != nil {
		resp.Error = "write failed"
		return resp
	}
	b, err := io.ReadAll(io.LimitReader(c, MaxResponse+1))
	if err != nil && len(b) == 0 {
		resp.Error = "no response"
		return resp
	}
	if len(b) > MaxResponse {
		b, resp.Truncated = b[:MaxResponse], true
	}
	resp.Text = strings.ToValidUTF8(strings.ReplaceAll(string(b), "\r\n", "\n"), "�")
	return resp
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	raw := q.String("query", "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	query, kind, err := Normalize(raw)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	res := Result{Query: query, Kind: kind, Chain: []Response{}}
	server := t.Root
	seen := map[string]bool{}
	for range MaxReferrals + 1 {
		if seen[strings.ToLower(server)] {
			break
		}
		seen[strings.ToLower(server)] = true
		resp := t.ask(r.Context(), server, queryFor(server, query, kind))
		res.Chain = append(res.Chain, resp)
		if resp.Error != "" {
			break
		}
		next := Referral(resp.Text)
		if next == "" {
			break
		}
		server = next
	}
	for i := len(res.Chain) - 1; i >= 0; i-- {
		if res.Chain[i].Text != "" {
			res.Parsed = Parse(res.Chain[i].Text)
			break
		}
	}
	if len(res.Chain) > 0 && res.Chain[0].Error != "" && len(res.Chain) == 1 {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "WHOIS server unreachable: " + res.Chain[0].Error})
		return
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: raw}})
}

var fieldPatterns = map[string][]*regexp.Regexp{
	"registrar": {regexp.MustCompile(`(?im)^\s*Registrar:\s*(.+)$`), regexp.MustCompile(`(?im)^\s*registrar:\s*(.+)$`)},
	"created":   {regexp.MustCompile(`(?im)^\s*(?:Creation Date|Created|created|RegDate|Registered on|registered):\s*(.+)$`)},
	"updated":   {regexp.MustCompile(`(?im)^\s*(?:Updated Date|Last Modified|last-modified|Updated|changed):\s*(.+)$`)},
	"expires":   {regexp.MustCompile(`(?im)^\s*(?:Registry Expiry Date|Registrar Registration Expiration Date|Expiry Date|Expiration Date|expires|paid-till):\s*(.+)$`)},
	"org":       {regexp.MustCompile(`(?im)^\s*(?:Registrant Organization|OrgName|org-name|organisation|Organization|owner|descr):\s*(.+)$`)},
	"country":   {regexp.MustCompile(`(?im)^\s*(?:Registrant Country|Country|country):\s*(\S+)`)},
	"netrange":  {regexp.MustCompile(`(?im)^\s*(?:NetRange|inetnum|inet6num|CIDR):\s*(.+)$`)},
}

var (
	statusRe = regexp.MustCompile(`(?im)^\s*(?:Domain )?Status:\s*(\S+)`)
	nsRe     = regexp.MustCompile(`(?im)^\s*(?:Name Server|nserver|nameserver):\s*(\S+)`)
)

func first(text, key string) string {
	for _, re := range fieldPatterns[key] {
		if m := re.FindStringSubmatch(text); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// Parse extracts common fields (best effort; WHOIS has no standard format).
func Parse(text string) Parsed {
	p := Parsed{
		Registrar: first(text, "registrar"), Created: first(text, "created"), Updated: first(text, "updated"),
		Expires: first(text, "expires"), Org: first(text, "org"), Country: first(text, "country"), NetRange: first(text, "netrange"),
		Status: []string{}, Nameservers: []string{},
	}
	seen := map[string]bool{}
	for _, m := range statusRe.FindAllStringSubmatch(text, -1) {
		if !seen[m[1]] {
			p.Status, seen[m[1]] = append(p.Status, m[1]), true
		}
	}
	for _, m := range nsRe.FindAllStringSubmatch(text, -1) {
		ns := strings.ToLower(strings.TrimSuffix(m[1], "."))
		if !seen[ns] {
			p.Nameservers, seen[ns] = append(p.Nameservers, ns), true
		}
	}
	return p
}
