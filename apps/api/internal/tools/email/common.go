// Package email implements EM-01 (MX + SMTP), EM-02 (SPF), EM-03 (DMARC) and EM-04 (DKIM).
package email

import (
	"context"
	"net/http"
	"strings"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools/apps/api/internal/target"
	"github.com/digibaro/net-tools/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools/apps/api/internal/tools/params"
)

// resolver is the DNS view all email checks use (the server's system resolver).
type resolver interface {
	TXT(ctx context.Context, name string) ([]string, string, error)
	MX(ctx context.Context, name string) ([]*dns.MX, string, error)
	Exists(ctx context.Context, name string, qtype uint16) (int, string, error)
}

type sysResolver struct{}

func (sysResolver) query(ctx context.Context, name string, qtype uint16) (*dns.Msg, error) {
	srv, err := dnsx.SystemServer(dnsx.UDP)
	if err != nil {
		return nil, err //nolint:wrapcheck // surfaced as INTERNAL
	}
	m, _, err := dnsx.Query(ctx, srv, name, qtype)
	if err != nil {
		return nil, dnslookup.Upstream(err)
	}
	return m, nil
}

// TXT returns the TXT strings (each record's chunks joined) and the rcode.
func (s sysResolver) TXT(ctx context.Context, name string) ([]string, string, error) {
	m, err := s.query(ctx, name, dns.TypeTXT)
	if err != nil {
		return nil, "", err
	}
	var out []string
	for _, rr := range m.Answer {
		if t, ok := rr.(*dns.TXT); ok {
			out = append(out, strings.Join(t.Txt, ""))
		}
	}
	return out, dnsx.RcodeName(m.Rcode), nil
}

// MX returns MX records sorted by preference.
func (s sysResolver) MX(ctx context.Context, name string) ([]*dns.MX, string, error) {
	m, err := s.query(ctx, name, dns.TypeMX)
	if err != nil {
		return nil, "", err
	}
	var out []*dns.MX
	for _, rr := range m.Answer {
		if mx, ok := rr.(*dns.MX); ok {
			out = append(out, mx)
		}
	}
	return out, dnsx.RcodeName(m.Rcode), nil
}

// Exists counts answers of qtype (used for SPF a/mx lookups and void-lookup accounting).
func (s sysResolver) Exists(ctx context.Context, name string, qtype uint16) (int, string, error) {
	m, err := s.query(ctx, name, qtype)
	if err != nil {
		return 0, "", err
	}
	n := 0
	for _, rr := range m.Answer {
		if rr.Header().Rrtype == qtype {
			n++
		}
	}
	return n, dnsx.RcodeName(m.Rcode), nil
}

func domainParam(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	q := params.New(r.URL.Query())
	raw := q.String(name, "", true, 253)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return "", false
	}
	d, err := target.NormalizeHostname(strings.TrimPrefix(strings.TrimSpace(raw), "@"))
	if err != nil || !strings.Contains(d, ".") {
		httpapi.Error(w, r, httpapi.InvalidInput(name, "enter a domain name such as example.com"))
		return "", false
	}
	return d, true
}

// tagList parses "k=v; k=v" (DMARC, DKIM).
func tagList(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	return out
}
