package email

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
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

// Relay test addresses use RFC 2606 domains; no DATA is ever sent, so nothing is delivered.
const (
	relayFrom    = "relay-test@nettools.example"
	relayTo      = "relay-test@example.net"
	relayTimeout = 30 * time.Second
)

// RelayCase is one MAIL FROM / RCPT TO attempt.
type RelayCase struct {
	Name      string `json:"name"`
	MailFrom  string `json:"mailFrom"`
	RcptTo    string `json:"rcptTo"`
	MailReply string `json:"mailReply"`
	RcptReply string `json:"rcptReply"`
	Accepted  bool   `json:"accepted"`
}

// RelayResult is the /email/relay response.
type RelayResult struct {
	Address  string      `json:"address"`
	Banner   string      `json:"banner"`
	StartTLS bool        `json:"startTls"`
	Cases    []RelayCase `json:"cases"`
	Open     bool        `json:"open"`
	Error    string      `json:"error"`
}

// RelayCases lists the attempts for a server whose own domain is domain (may be empty).
func RelayCases(domain string, ip netip.Addr) []RelayCase {
	cs := []RelayCase{
		{Name: "external", MailFrom: relayFrom, RcptTo: relayTo},
		{Name: "nullSender", MailFrom: "", RcptTo: relayTo},
		{Name: "percentHack", MailFrom: relayFrom, RcptTo: "relay-test%example.net@[" + ip.String() + "]"},
		{Name: "quoted", MailFrom: relayFrom, RcptTo: `"relay-test@example.net"`},
	}
	if domain != "" {
		cs = append(cs, RelayCase{Name: "spoofedLocal", MailFrom: "postmaster@" + domain, RcptTo: relayTo})
	}
	return cs
}

// smtpSession is a minimal client with bounded reads.
type smtpSession struct {
	c net.Conn
	r *bufio.Reader
}

func newSession(c net.Conn) *smtpSession {
	return &smtpSession{c: c, r: bufio.NewReader(io.LimitReader(c, MaxSMTPBytes))}
}

// reply reads one (possibly multi-line) reply and returns its code and text lines.
func (s *smtpSession) reply() (int, []string, error) {
	var lines []string
	for len(lines) < MaxReplyLines {
		l, err := s.r.ReadString('\n')
		if err != nil {
			return 0, lines, fmt.Errorf("read reply: %w", err)
		}
		l = strings.TrimRight(l, "\r\n")
		if len(l) < 3 {
			return 0, lines, fmt.Errorf("malformed reply %q", l)
		}
		code, err := strconv.Atoi(l[:3])
		if err != nil {
			return 0, lines, fmt.Errorf("malformed reply %q", l)
		}
		lines = append(lines, l[min(4, len(l)):])
		if len(l) < 4 || l[3] != '-' {
			return code, lines, nil
		}
	}
	return 0, lines, errors.New("reply too long")
}

func (s *smtpSession) cmd(line string) (int, []string, error) {
	if _, err := fmt.Fprintf(s.c, "%s\r\n", line); err != nil {
		return 0, nil, fmt.Errorf("write: %w", err)
	}
	return s.reply()
}

func short(code int, lines []string) string {
	text := strings.Join(lines, " ")
	if len(text) > 200 {
		text = text[:200] + "…"
	}
	return strconv.Itoa(code) + " " + text
}

// RunRelayTest performs the attempts on an open connection (banner not yet read).
func RunRelayTest(ctx context.Context, c net.Conn, serverName string, cases []RelayCase) RelayResult {
	res := RelayResult{Cases: []RelayCase{}}
	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetDeadline(dl)
	}
	s := newSession(c)
	code, banner, err := s.reply()
	if err != nil || code != 220 {
		res.Error = "no SMTP banner"
		return res
	}
	res.Banner = strings.Join(banner, " ")
	code, ext, err := s.cmd("EHLO nettools.invalid")
	if err != nil || code != 250 {
		res.Error = "EHLO rejected"
		return res
	}
	for _, e := range ext[1:] {
		if strings.EqualFold(strings.Fields(e + " ")[0], "STARTTLS") {
			res.StartTLS = true
		}
	}
	if res.StartTLS {
		if code, _, err := s.cmd("STARTTLS"); err == nil && code == 220 {
			tc := tls.Client(c, &tls.Config{ServerName: serverName, InsecureSkipVerify: true, MinVersion: tls.VersionTLS10}) //nolint:gosec // relay test only; certificates are checked by the MX tool
			if err := tc.HandshakeContext(ctx); err != nil {
				res.Error = "TLS handshake failed after STARTTLS"
				return res
			}
			s = newSession(tc)
			if code, _, err := s.cmd("EHLO nettools.invalid"); err != nil || code != 250 {
				res.Error = "EHLO rejected after STARTTLS"
				return res
			}
		}
	}
	for _, rc := range cases {
		code, lines, err := s.cmd("MAIL FROM:<" + rc.MailFrom + ">")
		if err != nil {
			res.Error = "connection lost"
			break
		}
		rc.MailReply = short(code, lines)
		if code/100 == 2 {
			code, lines, err = s.cmd("RCPT TO:<" + rc.RcptTo + ">")
			if err != nil {
				res.Error = "connection lost"
				res.Cases = append(res.Cases, rc)
				break
			}
			rc.RcptReply = short(code, lines)
			rc.Accepted = code/100 == 2
			res.Open = res.Open || rc.Accepted
		}
		res.Cases = append(res.Cases, rc)
		if _, _, err := s.cmd("RSET"); err != nil { // never DATA: the transaction is always reset
			res.Error = "connection lost"
			break
		}
	}
	_, _ = fmt.Fprint(s.c, "QUIT\r\n")
	return res
}

// RelayTool is EM-09.
type RelayTool struct {
	deps tools.Deps
	Dial func(ctx context.Context, addr string) (net.Conn, error)
}

func (*RelayTool) ID() string { return "email.relay" }
func (*RelayTool) Enabled(d tools.Deps) (bool, tools.Reason) {
	return tools.RequireFlag(d.Config.Features.OpenRelayTest)
}

func (t *RelayTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Dial == nil {
		t.Dial = func(ctx context.Context, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 8 * time.Second}
			return d.DialContext(ctx, "tcp", addr) //nolint:wrapcheck // classified below
		}
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(relayTimeout+5*time.Second)).Get("/email/relay", t.handle)
}

func (t *RelayTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	host := q.String("host", "", true, 253)
	port := q.Int("port", 25, 1, 65535)
	domain := q.String("domain", "", false, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if port != 25 && port != 587 && port != 2525 {
		httpapi.Error(w, r, httpapi.InvalidInput("port", "the relay test uses port 25, 587 or 2525"))
		return
	}
	if domain != "" {
		d, err := target.NormalizeHostname(strings.TrimSpace(domain))
		if err != nil || !strings.Contains(d, ".") {
			httpapi.Error(w, r, httpapi.InvalidInput("domain", "enter the mail domain, such as example.com"))
			return
		}
		domain = d
	}
	tgt, err := t.deps.Target.Resolve(r.Context(), host, target.Opts{Field: "host"})
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), relayTimeout)
	defer cancel()
	addr := netip.AddrPortFrom(tgt.IP, uint16(port)).String() //nolint:gosec // 25, 587 or 2525
	res := RelayResult{Address: addr, Cases: []RelayCase{}}
	c, err := t.Dial(ctx, addr)
	if err != nil {
		res.Error = "cannot connect to port " + strconv.Itoa(port) + " (outbound SMTP is often blocked by the server's network provider)"
	} else {
		defer c.Close()
		res = RunRelayTest(ctx, c, strings.TrimSuffix(host, "."), RelayCases(domain, tgt.IP))
		res.Address = addr
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}
