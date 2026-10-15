// Package revocation implements TLS-04: is a server's certificate revoked? It checks the
// stapled OCSP response, the CA's OCSP responder and the CRL distribution points.
package revocation

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/ocsp"

	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/netx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tlsinspect"
)

// Limits.
const (
	MaxCRLBytes  = 20 << 20 // large CAs publish CRLs of several MB
	FetchTimeout = 15 * time.Second
	MaxURLs      = 3
)

// Tool is tls.revocation.
type Tool struct {
	deps tools.Deps
	Now  func() time.Time
}

func (*Tool) ID() string                                { return "tls.revocation" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	if t.Now == nil {
		t.Now = time.Now
	}
	r.With(d.RateLimit(limits.Heavy), d.Slot, d.Timeout(60*time.Second)).Get("/tls/revocation", t.handle)
}

// OCSPResult is one OCSP check (stapled or from a responder).
type OCSPResult struct {
	Source     string `json:"source"`
	Status     string `json:"status"`
	ThisUpdate string `json:"thisUpdate"`
	NextUpdate string `json:"nextUpdate"`
	RevokedAt  string `json:"revokedAt"`
	Reason     string `json:"reason"`
	Error      string `json:"error"`
}

// CRLResult is one CRL check.
type CRLResult struct {
	URL        string `json:"url"`
	Status     string `json:"status"`
	Entries    int    `json:"entries"`
	ThisUpdate string `json:"thisUpdate"`
	NextUpdate string `json:"nextUpdate"`
	RevokedAt  string `json:"revokedAt"`
	Reason     string `json:"reason"`
	Error      string `json:"error"`
}

// Result is the TLS-04 payload. Status: good, revoked or unknown (no source could answer).
type Result struct {
	Host    string       `json:"host"`
	Subject string       `json:"subject"`
	Serial  string       `json:"serial"`
	Issuer  string       `json:"issuer"`
	Status  string       `json:"status"`
	OCSP    []OCSPResult `json:"ocsp"`
	CRL     []CRLResult  `json:"crl"`
}

var reasons = map[int]string{0: "unspecified", 1: "key compromise", 2: "CA compromise", 3: "affiliation changed", 4: "superseded", 5: "cessation of operation", 6: "certificate hold", 8: "remove from CRL", 9: "privilege withdrawn", 10: "AA compromise"}

func ts(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func ocspStatus(s int) string {
	switch s {
	case ocsp.Good:
		return "good"
	case ocsp.Revoked:
		return "revoked"
	}
	return "unknown"
}

func fromOCSP(source string, resp *ocsp.Response) OCSPResult {
	o := OCSPResult{Source: source, Status: ocspStatus(resp.Status), ThisUpdate: ts(resp.ThisUpdate), NextUpdate: ts(resp.NextUpdate)}
	if resp.Status == ocsp.Revoked {
		o.RevokedAt, o.Reason = ts(resp.RevokedAt), reasons[resp.RevocationReason]
	}
	return o
}

// get fetches a URL taken from certificate data through the policy client (redirects included).
func (t *Tool) get(ctx context.Context, u string, limit int64) ([]byte, error) {
	pu, err := url.Parse(u)
	if err != nil || (pu.Scheme != "http" && pu.Scheme != "https") {
		return nil, errors.New("not an http(s) URL")
	}
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil) //nolint:gosec // URL from certificate data; dialed only through PolicyClient (TestRevocationURLPolicy)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	resp, err := netx.PolicyClient(t.deps.Target, t.deps.UserAgent, FetchTimeout).Do(req) //nolint:gosec // every dial and redirect is policy-checked
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return nil, fmt.Errorf("fetch: %s", httpapi.ToAPIError(err).Message)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("larger than %d MB", limit>>20)
	}
	return b, nil
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	host := q.String("host", "", true, 260)
	port := q.Int("port", 443, 1, 65535)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	s, err := tlsinspect.Inspect(ctx, t.deps.Target, host, port, "", "")
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	if len(s.Certs) < 2 {
		httpapi.Error(w, r, &httpapi.APIError{Code: httpapi.CodeInvalidInput, Message: "the server sent no issuer certificate; revocation cannot be checked", Details: map[string]any{"field": "host"}})
		return
	}
	leaf, issuer := s.Certs[0], s.Certs[1]
	out := Result{Host: host, Subject: leaf.Subject.CommonName, Serial: leaf.SerialNumber.Text(16), Issuer: issuer.Subject.CommonName, OCSP: []OCSPResult{}, CRL: []CRLResult{}}
	out.OCSP, out.CRL = t.check(ctx, leaf, issuer, s.OCSPResponse)
	out.Status = verdict(out)
	httpapi.JSON(w, r, out, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: host, Used: s.ConnectedTo}})
}

// check runs every available revocation source for leaf.
func (t *Tool) check(ctx context.Context, leaf, issuer *x509.Certificate, stapled []byte) ([]OCSPResult, []CRLResult) {
	ocsps, crls := []OCSPResult{}, []CRLResult{}
	if len(stapled) > 0 {
		if resp, err := ocsp.ParseResponseForCert(stapled, leaf, issuer); err == nil {
			ocsps = append(ocsps, fromOCSP("stapled", resp))
		} else {
			ocsps = append(ocsps, OCSPResult{Source: "stapled", Status: "unknown", Error: "invalid stapled response"})
		}
	}
	for i, server := range leaf.OCSPServer {
		if i >= MaxURLs {
			break
		}
		req, err := ocsp.CreateRequest(leaf, issuer, &ocsp.RequestOptions{})
		if err != nil {
			ocsps = append(ocsps, OCSPResult{Source: server, Status: "unknown", Error: "cannot build request"})
			continue
		}
		// RFC 6960 GET form: base64 request appended to the responder URL.
		u := strings.TrimSuffix(server, "/") + "/" + url.PathEscape(base64.StdEncoding.EncodeToString(req))
		b, err := t.get(ctx, u, 64<<10)
		if err != nil {
			ocsps = append(ocsps, OCSPResult{Source: server, Status: "unknown", Error: err.Error()})
			continue
		}
		resp, err := ocsp.ParseResponseForCert(b, leaf, issuer)
		if err != nil {
			ocsps = append(ocsps, OCSPResult{Source: server, Status: "unknown", Error: "invalid response: " + err.Error()})
			continue
		}
		ocsps = append(ocsps, fromOCSP(server, resp))
	}
	for i, u := range leaf.CRLDistributionPoints {
		if i >= MaxURLs {
			break
		}
		crls = append(crls, t.checkCRL(ctx, u, leaf.SerialNumber, issuer))
	}
	return ocsps, crls
}

func (t *Tool) checkCRL(ctx context.Context, u string, serial *big.Int, issuer *x509.Certificate) CRLResult {
	c := CRLResult{URL: u, Status: "unknown"}
	b, err := t.get(ctx, u, MaxCRLBytes)
	if err != nil {
		c.Error = err.Error()
		return c
	}
	rl, err := x509.ParseRevocationList(b)
	if err != nil {
		c.Error = "not a valid CRL"
		return c
	}
	if err := rl.CheckSignatureFrom(issuer); err != nil {
		c.Error = "CRL signature does not verify against the issuer"
		return c
	}
	c.Entries, c.ThisUpdate, c.NextUpdate = len(rl.RevokedCertificateEntries), ts(rl.ThisUpdate), ts(rl.NextUpdate)
	c.Status = "good"
	for _, e := range rl.RevokedCertificateEntries {
		if e.SerialNumber.Cmp(serial) == 0 {
			c.Status, c.RevokedAt, c.Reason = "revoked", ts(e.RevocationTime), reasons[e.ReasonCode]
			break
		}
	}
	if !rl.NextUpdate.IsZero() && rl.NextUpdate.Before(t.Now()) {
		c.Error = "the CRL is past its next update time (stale)"
	}
	return c
}

func verdict(r Result) string {
	good := false
	for _, o := range r.OCSP {
		if o.Status == "revoked" {
			return "revoked"
		}
		good = good || o.Status == "good"
	}
	for _, c := range r.CRL {
		if c.Status == "revoked" {
			return "revoked"
		}
		good = good || c.Status == "good"
	}
	if good {
		return "good"
	}
	return "unknown"
}
