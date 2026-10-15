// Package tlsinspect implements TLS-01 (certificate inspector) and TLS-02 (expiry checker).
package tlsinspect

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/target"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tlsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// InspectTool is TLS-01.
type InspectTool struct{ deps tools.Deps }

func (*InspectTool) ID() string                                { return "tls.inspect" }
func (*InspectTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *InspectTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(20*time.Second)).Get("/tls/inspect", t.handle)
}

func handshakeErr(err error) error {
	msg := err.Error()
	for _, k := range []string{"connection refused", "i/o timeout", "no route", "unexpected reply", "STARTTLS refused", "STLS refused", "handshake failure", "protocol version", "EOF", "connection reset"} {
		if strings.Contains(msg, k) {
			return &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "TLS connection failed: " + k, Err: err}
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &httpapi.APIError{Code: httpapi.CodeTimeout, Message: "the TLS handshake timed out", Err: err}
	}
	return &httpapi.APIError{Code: httpapi.CodeProviderError, Message: "TLS connection failed", Err: err}
}

// Inspect resolves host (policy), dials the pinned address and handshakes.
func Inspect(ctx context.Context, res *target.Resolver, host string, port int, sni, starttls string) (*tlsx.Session, error) {
	tgt, err := res.Resolve(ctx, host, target.Opts{AllowPort: true, Field: "host"})
	if err != nil {
		return nil, err //nolint:wrapcheck // client-facing
	}
	if tgt.Port != 0 {
		port = tgt.Port
	}
	if sni == "" && !tgt.IsIP {
		sni = tgt.Host
	}
	if sni == "" {
		sni = tgt.IP.String()
	}
	s, err := tlsx.Handshake(ctx, tgt.AddrPort(port), tlsx.Options{ServerName: sni, StartTLS: starttls, ALPN: []string{"h2", "http/1.1"}})
	if err != nil {
		return nil, handshakeErr(err)
	}
	return s, nil
}

func (t *InspectTool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	host := q.String("host", "", true, 300)
	port := q.Int("port", 443, 1, 65535)
	sni := q.String("sni", "", false, 253)
	starttls := q.Enum("starttls", "", tlsx.StartTLSProtocols...)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if sni != "" {
		n, err := target.NormalizeHostname(sni)
		if err != nil {
			httpapi.Error(w, r, httpapi.InvalidInput("sni", "not a valid host name"))
			return
		}
		sni = n
	}
	s, err := Inspect(r.Context(), t.deps.Target, host, port, sni, starttls)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	httpapi.JSON(w, r, s, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: host, Used: s.ConnectedTo}})
}

// ExpiryTool is TLS-02.
type ExpiryTool struct{ deps tools.Deps }

func (*ExpiryTool) ID() string                                { return "tls.expiry" }
func (*ExpiryTool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *ExpiryTool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(60*time.Second)).Post("/tls/expiry", t.handle)
}

// ExpiryRow is one target's result.
type ExpiryRow struct {
	Target      string `json:"target"`
	Address     string `json:"address"`
	NotAfter    string `json:"notAfter"`
	DaysLeft    *int   `json:"daysLeft"`
	Issuer      string `json:"issuer"`
	Subject     string `json:"subject"`
	Verified    bool   `json:"verified"`
	VerifyError string `json:"verifyError"`
	Error       string `json:"error"`
	ErrorCode   string `json:"errorCode"`
}

func (t *ExpiryTool) handle(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Targets []string `json:"targets"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		httpapi.Error(w, r, httpapi.InvalidInput("targets", `expected JSON {"targets": ["host[:port]", …]}`))
		return
	}
	limit := t.deps.Config.Limits.BulkMaxTargets
	if len(in.Targets) == 0 || len(in.Targets) > limit {
		httpapi.Error(w, r, httpapi.InvalidInput("targets", "between 1 and BULK_MAX_TARGETS targets"))
		return
	}
	rows := make([]ExpiryRow, len(in.Targets))
	ctx := r.Context()
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, raw := range in.Targets {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			rows[i] = expiry(ctx, t.deps.Target, strings.TrimSpace(raw))
		})
	}
	wg.Wait()
	httpapi.JSON(w, r, map[string]any{"rows": rows}, &httpapi.Meta{Tool: t.ID(), Vantage: "server"})
}

func expiry(ctx context.Context, res *target.Resolver, raw string) ExpiryRow {
	row := ExpiryRow{Target: raw}
	s, err := Inspect(ctx, res, raw, 443, "", "")
	if err != nil {
		a := httpapi.ToAPIError(err)
		row.Error, row.ErrorCode = a.Message, string(a.Code)
		return row
	}
	row.Address, row.Verified, row.VerifyError = s.ConnectedTo, s.Verified, s.VerifyError
	if len(s.Chain) > 0 {
		leaf := s.Chain[0]
		days := leaf.DaysLeft
		row.NotAfter, row.DaysLeft, row.Issuer, row.Subject = leaf.NotAfter, &days, leaf.Issuer, leaf.Subject
	}
	return row
}
