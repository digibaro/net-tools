package svcb

import (
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// echList is an ECHConfigList with one X25519 config for public name "public.example".
func echList() []byte {
	pk := make([]byte, 32)
	body := []byte{7, 0x00, 0x20, 0, 32}
	body = append(body, pk...)
	body = append(body, 0, 8, 0, 1, 0, 1, 0, 1, 0, 3, 14)
	body = append(body, byte(len("public.example")))
	body = append(body, "public.example"...)
	body = append(body, 0, 0)
	cfg := append([]byte{0xfe, 0x0d, byte(len(body) >> 8), byte(len(body))}, body...)
	return append([]byte{byte(len(cfg) >> 8), byte(len(cfg))}, cfg...)
}

func TestParseECH(t *testing.T) {
	cfgs, err := ParseECHConfigList(echList())
	if err != nil || len(cfgs) != 1 {
		t.Fatalf("%v %v", cfgs, err)
	}
	c := cfgs[0]
	if c.ConfigID != 7 || c.KEM != "DHKEM(X25519, HKDF-SHA256)" || c.PublicName != "public.example" || len(c.Suites) != 2 || c.Suites[1].AEAD != "ChaCha20Poly1305" || c.MaxNameLength != 14 {
		t.Errorf("cfg = %+v", c)
	}
	if _, err := ParseECHConfigList(echList()[:20]); err == nil {
		t.Error("truncated list accepted")
	}
}

func TestHTTPS(t *testing.T) {
	ech := base64.StdEncoding.EncodeToString(echList())
	addr := dnstest.Start(t, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		var s string
		switch r.Question[0].Name {
		case "example.test.":
			s = `example.test. 300 IN HTTPS 0 cdn.example.net.`
		case "cdn.example.net.":
			s = `cdn.example.net. 300 IN HTTPS 1 . alpn="h3,h2" ipv4hint=192.0.2.1 ech="` + ech + `"`
		}
		if s != "" {
			rr, err := dns.NewRR(s)
			if err != nil {
				t.Error(err)
			}
			m.Answer = append(m.Answer, rr)
		}
		_ = w.WriteMsg(m)
	})
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/svcb?name=example.test&resolver="+url.QueryEscape(addr.String())).Must(t).Decode(t, &res)
	if len(res.Records) != 2 || res.Records[0].Mode != "alias" || res.Aliases[0] != "cdn.example.net" {
		t.Fatalf("res = %+v", res)
	}
	svc := res.Records[1]
	if svc.Priority != 1 || svc.Params[0].Key != "alpn" || svc.Params[0].Value != "h3,h2" || len(svc.ECH) != 1 || svc.ECH[0].PublicName != "public.example" {
		t.Errorf("service = %+v", svc)
	}
	env.Get("/api/v1/dns/svcb?name=example.test&type=MX").ExpectError(t, "INVALID_INPUT")
}
