package web

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/quic-go/quic-go"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestParseAltSvc(t *testing.T) {
	got := ParseAltSvc(`h3=":443"; ma=86400, h3-29=":8443", h2="alt.example:443"; ma=60, clear`)
	if len(got) != 3 || got[0].Protocol != "h3" || got[0].Port != 443 || got[0].MaxAge != 86400 || got[2].Host != "alt.example" || got[2].MaxAge != 60 {
		t.Errorf("got %+v", got)
	}
}

func TestProtocols(t *testing.T) {
	var quicPort int
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Alt-Svc", `h3=":`+strconv.Itoa(quicPort)+`"; ma=3600`)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	// A QUIC listener with the same certificate, offering ALPN h3.
	ln, err := quic.ListenAddr("127.0.0.1:0", &tls.Config{Certificates: srv.TLS.Certificates, NextProtos: []string{"h3"}, MinVersion: tls.VersionTLS13}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	quicPort = ln.Addr().(*net.UDPAddr).Port
	go func() {
		for {
			c, err := ln.Accept(context.Background())
			if err != nil {
				return
			}
			go func() {
				<-c.Context().Done()
			}()
		}
	}()

	env := tooltest.New(t, &ProtocolsTool{}, tooltest.Options{AllowPrivate: true})
	var out ProtocolsResult
	env.Get("/api/v1/http/protocols?url="+url.QueryEscape(srv.URL+"/")).Must(t).Decode(t, &out)
	if !out.HTTP2 || out.ALPN != "h2" || out.TLSVersion == "" || len(out.AltSvc) != 1 || !out.HTTP3.Advertised || out.HTTP3.Port != quicPort || !out.HTTP3.OK || out.HTTP3.QUICVersion == "" {
		t.Errorf("out = %+v", out)
	}

	// Without a QUIC listener on the forced port, HTTP/3 is reported as unavailable.
	var noH3 ProtocolsResult
	tooltest.New(t, &ProtocolsTool{H3Port: 9}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/http/protocols?url="+url.QueryEscape(srv.URL+"/")).Must(t).Decode(t, &noH3)
	if noH3.HTTP3.OK || noH3.HTTP3.Error == "" {
		t.Errorf("no h3 = %+v", noH3.HTTP3)
	}

	strict := tooltest.New(t, &ProtocolsTool{}, tooltest.Options{})
	strict.Get("/api/v1/http/protocols?url="+url.QueryEscape("https://127.0.0.1/")).ExpectError(t, "TARGET_PRIVATE_DENIED")
	strict.Get("/api/v1/http/protocols?url="+url.QueryEscape("http://example.com/")).ExpectError(t, "INVALID_INPUT")
}
