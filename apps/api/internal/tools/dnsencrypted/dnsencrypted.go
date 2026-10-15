// Package dnsencrypted implements DNS-06: test a DoT or DoH endpoint (TLS details + latency).
package dnsencrypted

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx"
	"github.com/digibaro/net-tools-docker/apps/api/internal/httpapi"
	"github.com/digibaro/net-tools-docker/apps/api/internal/limits"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/dnslookup"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/params"
)

// Tool is the encrypted DNS tester.
type Tool struct {
	deps tools.Deps
	// RootCAs overrides system roots (tests).
	RootCAs *x509.CertPool
}

func (*Tool) ID() string                                { return "dns.encrypted" }
func (*Tool) Enabled(d tools.Deps) (bool, tools.Reason) { return tools.Always(d) }

func (t *Tool) Mount(r chi.Router, d tools.Deps) {
	t.deps = d
	r.With(d.RateLimit(limits.Light), d.Slot, d.Timeout(30*time.Second)).Get("/dns/encrypted", t.handle)
}

// TLSInfo describes the handshake with the endpoint.
type TLSInfo struct {
	Version     string   `json:"version"`
	Cipher      string   `json:"cipher"`
	ALPN        string   `json:"alpn"`
	Subject     string   `json:"subject"`
	Issuer      string   `json:"issuer"`
	NotAfter    string   `json:"notAfter"`
	DNSNames    []string `json:"dnsNames"`
	Verified    bool     `json:"verified"`
	VerifyError string   `json:"verifyError"`
	HandshakeMs float64  `json:"handshakeMs"`
}

// Query is one timed query.
type Query struct {
	RTTMs float64 `json:"rttMs"`
	Rcode string  `json:"rcode"`
	Error string  `json:"error"`
}

// Result is the response payload.
type Result struct {
	Endpoint  string        `json:"endpoint"`
	Transport string        `json:"transport"`
	Address   string        `json:"address"`
	TLS       *TLSInfo      `json:"tls"`
	Queries   []Query       `json:"queries"`
	Answer    []dnsx.Record `json:"answer"`
	MinMs     float64       `json:"minMs"`
	AvgMs     float64       `json:"avgMs"`
	MaxMs     float64       `json:"maxMs"`
}

func (t *Tool) handle(w http.ResponseWriter, r *http.Request) {
	q := params.New(r.URL.Query())
	endpoint := q.String("endpoint", "", true, 300)
	transport := q.Enum("transport", "dot", "dot", "doh")
	name := q.String("name", "example.com", false, 253)
	typ := q.Enum("type", "A", dnslookup.RecordTypes...)
	count := q.Int("count", 3, 1, 5)
	if err := q.Err(); err != nil {
		httpapi.Error(w, r, err)
		return
	}
	qtype, _ := dnsx.ParseType(typ)
	qname, err := dnslookup.ParseName(name, qtype)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	ctx := r.Context()
	srv, err := dnsx.ResolveServer(ctx, t.deps.Target, endpoint, dnsx.Transport(transport), 0)
	if err != nil {
		httpapi.Error(w, r, err)
		return
	}
	srv.RootCAs = t.RootCAs
	res := Result{Endpoint: srv.Label, Transport: transport, Address: srv.Addr.String(), Queries: []Query{}, Answer: []dnsx.Record{}}
	info, err := t.handshake(ctx, srv)
	if err != nil {
		httpapi.Error(w, r, dnslookup.Upstream(err))
		return
	}
	res.TLS = info
	var rtts []float64
	for range count {
		msg, rtt, err := dnsx.Exchange(ctx, srv, dnsx.NewQuery(qname, qtype, false))
		if err != nil {
			res.Queries = append(res.Queries, Query{Error: "query failed"})
			continue
		}
		ms := dnslookup.Ms(rtt)
		rtts = append(rtts, ms)
		res.Queries = append(res.Queries, Query{RTTMs: ms, Rcode: dnsx.RcodeName(msg.Rcode)})
		res.Answer = dnsx.Records(msg.Answer)
	}
	if len(rtts) > 0 {
		res.MinMs, res.MaxMs = slices.Min(rtts), slices.Max(rtts)
		sum := 0.0
		for _, v := range rtts {
			sum += v
		}
		res.AvgMs = float64(int(sum/float64(len(rtts))*1000)) / 1000
	}
	httpapi.JSON(w, r, res, &httpapi.Meta{Tool: t.ID(), Vantage: "server", Target: &httpapi.TargetMeta{Input: endpoint, Used: srv.Addr.String()}})
}

// handshake connects to the pinned address and reports TLS details; verification failures are
// reported, not fatal, so misconfigured endpoints can still be inspected.
func (t *Tool) handshake(ctx context.Context, srv dnsx.Server) (*TLSInfo, error) {
	alpn := []string{"dot"}
	if srv.Transport == dnsx.DoH {
		alpn = []string{"h2", "http/1.1"}
	}
	d := &net.Dialer{Timeout: 5 * time.Second}
	start := time.Now()
	raw, err := d.DialContext(ctx, "tcp", srv.Addr.String())
	if err != nil {
		return nil, err //nolint:wrapcheck // mapped by Upstream
	}
	defer raw.Close()
	name := srv.TLSName
	if name == "" {
		name = srv.Addr.Addr().String()
	}
	conn := tls.Client(raw, &tls.Config{ServerName: name, NextProtos: alpn, InsecureSkipVerify: true, MinVersion: tls.VersionTLS10}) //nolint:gosec // verified manually below
	hctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.HandshakeContext(hctx); err != nil {
		return nil, err //nolint:wrapcheck // mapped by Upstream
	}
	st := conn.ConnectionState()
	info := &TLSInfo{Version: tls.VersionName(st.Version), Cipher: tls.CipherSuiteName(st.CipherSuite), ALPN: st.NegotiatedProtocol, HandshakeMs: dnslookup.Ms(time.Since(start)), DNSNames: []string{}}
	if len(st.PeerCertificates) == 0 {
		info.VerifyError = "no certificate presented"
		return info, nil
	}
	leaf := st.PeerCertificates[0]
	info.Subject, info.Issuer, info.NotAfter, info.DNSNames = leaf.Subject.String(), leaf.Issuer.String(), leaf.NotAfter.UTC().Format(time.RFC3339), leaf.DNSNames
	inter := x509.NewCertPool()
	for _, c := range st.PeerCertificates[1:] {
		inter.AddCert(c)
	}
	opts := x509.VerifyOptions{DNSName: srv.TLSName, Intermediates: inter, Roots: t.RootCAs}
	if srv.TLSName == "" {
		opts.DNSName = srv.Addr.Addr().String()
	}
	if _, err := leaf.Verify(opts); err != nil {
		var hn x509.HostnameError
		if errors.As(err, &hn) {
			info.VerifyError = "certificate does not match " + opts.DNSName
		} else {
			info.VerifyError = err.Error()
		}
	} else {
		info.Verified = true
	}
	return info, nil
}
