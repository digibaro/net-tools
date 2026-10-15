package web

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

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

	// An HTTP/3 server with the same certificate that accepts 0-RTT.
	pc, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	quicPort = pc.LocalAddr().(*net.UDPAddr).Port
	h3srv := &http3.Server{
		Handler:    http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Server", "test-h3") }),
		TLSConfig:  http3.ConfigureTLSConfig(&tls.Config{Certificates: srv.TLS.Certificates, MinVersion: tls.VersionTLS13}),
		QUICConfig: &quic.Config{Allow0RTT: true},
	}
	go func() { _ = h3srv.Serve(pc) }()
	defer h3srv.Close()

	env := tooltest.New(t, &ProtocolsTool{}, tooltest.Options{AllowPrivate: true})
	var out ProtocolsResult
	env.Get("/api/v1/http/protocols?url="+url.QueryEscape(srv.URL+"/")).Must(t).Decode(t, &out)
	if !out.HTTP2 || out.ALPN != "h2" || out.TLSVersion == "" || len(out.AltSvc) != 1 || !out.HTTP3.Advertised || out.HTTP3.Port != quicPort || !out.HTTP3.OK || out.HTTP3.QUICVersion == "" || out.TTFBMs <= 0 {
		t.Errorf("out = %+v", out)
	}
	if r := out.HTTP3.Request; r == nil || !r.OK || r.Status != 200 || r.Server != "test-h3" || !r.Resumed || !r.ZeroRTT {
		t.Errorf("h3 request = %+v", r)
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
