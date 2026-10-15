package openresolver

import (
	"net/url"
	"testing"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/dnsx/dnstest"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func resolver(open bool) func(dns.ResponseWriter, *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		if !open {
			m.Rcode = dns.RcodeRefused
			_ = w.WriteMsg(m)
			return
		}
		m.RecursionAvailable = true
		if r.Question[0].Qtype == dns.TypeDNSKEY {
			for range 4 {
				rr, _ := dns.NewRR(". 172800 IN DNSKEY 257 3 8 AwEAAaz/tAm8yTn4Mfeh5eyI96WSVexTBAvkMgJzkKTOiW1vkIbzxeF3+/4RgWOq7HrxRixHlFlExOLAJr5emLvN7SWXgnLh4+B5xQlNVz8Og8kvArMtNROxVQuCaSnIDdD5LKyWbRd2n9WGe2R8PzgCmr3EgVLrjyBxWezF0jLHwVN8efS3rCj/EWgvIWgb9tarpVUDK/b58Da+sqqls3eNbuv7pr+eoZG+SrDK6nWeL3c6H5Apxz7LjVc1uTIdsIXxuOLYA4/ilBmSVIzuDWfdRUfhHdY6+cn8HFRm+2hM8AnXGXws9555KrUB5qihylGa8subX2Nn6UwNR1AkUTV74bU=")
				m.Answer = append(m.Answer, rr)
			}
		} else {
			rr, _ := dns.NewRR("example.com. 300 IN A 192.0.2.10")
			m.Answer = append(m.Answer, rr)
		}
		if o := r.IsEdns0(); o != nil {
			m.SetEdns0(4096, o.Do())
		}
		_ = w.WriteMsg(m)
	}
}

func TestOpenResolver(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/dns/open-resolver?server="+url.QueryEscape(dnstest.Start(t, resolver(true)).String())).Must(t).Decode(t, &res)
	if !res.Responds || !res.Open || res.Amplification == nil || res.Amplification.Factor < 10 {
		t.Errorf("open = %+v %+v", res, res.Amplification)
	}
	env.Get("/api/v1/dns/open-resolver?server="+url.QueryEscape(dnstest.Start(t, resolver(false)).String())).Must(t).Decode(t, &res)
	if !res.Responds || res.Open || res.Rcode != "REFUSED" || res.Amplification != nil {
		t.Errorf("closed = %+v", res)
	}
	tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/dns/open-resolver?server=192.168.1.1").ExpectError(t, "TARGET_PRIVATE_DENIED")
}
