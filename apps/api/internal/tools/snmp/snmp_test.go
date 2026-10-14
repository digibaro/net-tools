package snmp

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"

	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

// mib is the fake agent's tree, in lexicographic OID order.
var mib = []gosnmp.SnmpPDU{
	{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: []byte("Linux router 6.1")},
	{Name: ".1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: uint32(123456)},
	{Name: ".1.3.6.1.2.1.1.5.0", Type: gosnmp.OctetString, Value: []byte("gw")},
	{Name: ".1.3.6.1.2.1.2.2.1.6.1", Type: gosnmp.OctetString, Value: []byte{0, 0x11, 0x22, 0x33, 0x44, 0x55}},
	{Name: ".1.3.6.1.2.1.2.2.1.10.1", Type: gosnmp.Counter32, Value: uint32(4000000000)},
	{Name: ".1.3.6.1.2.1.31.1.1.1.6.1", Type: gosnmp.Counter64, Value: uint64(18000000000000000000)},
}

func oidLess(a, b string) bool {
	pa, pb := strings.Split(strings.Trim(a, "."), "."), strings.Split(strings.Trim(b, "."), ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x < y
		}
	}
	return len(pa) < len(pb)
}

func next(oid string) (gosnmp.SnmpPDU, bool) {
	for _, p := range mib {
		if oidLess(oid, p.Name) {
			return p, true
		}
	}
	return gosnmp.SnmpPDU{}, false
}

// agent is a minimal SNMPv2c responder (Get, GetNext, GetBulk) for community "public".
func agent(t *testing.T) netip.AddrPort {
	t.Helper()
	pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	dec := &gosnmp.GoSNMP{Version: gosnmp.Version2c}
	go func() {
		buf := make([]byte, 65535)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			req, err := dec.SnmpDecodePacket(buf[:n])
			if err != nil || req.Community != "public" {
				continue // wrong community: stay silent, like real agents
			}
			resp := &gosnmp.SnmpPacket{Version: gosnmp.Version2c, Community: "public", PDUType: gosnmp.GetResponse, RequestID: req.RequestID}
			switch req.PDUType {
			case gosnmp.GetRequest:
				for _, v := range req.Variables {
					i := slices.IndexFunc(mib, func(p gosnmp.SnmpPDU) bool { return p.Name == v.Name })
					if i < 0 {
						resp.Variables = append(resp.Variables, gosnmp.SnmpPDU{Name: v.Name, Type: gosnmp.NoSuchObject})
						continue
					}
					resp.Variables = append(resp.Variables, mib[i])
				}
			case gosnmp.GetNextRequest, gosnmp.GetBulkRequest:
				reps := 1
				if req.PDUType == gosnmp.GetBulkRequest {
					reps = int(req.MaxRepetitions)
				}
				cur := req.Variables[0].Name
				for range reps {
					p, ok := next(cur)
					if !ok {
						resp.Variables = append(resp.Variables, gosnmp.SnmpPDU{Name: cur, Type: gosnmp.EndOfMibView})
						break
					}
					resp.Variables = append(resp.Variables, p)
					cur = p.Name
				}
			}
			b, err := resp.MarshalMsg()
			if err == nil {
				_, _ = pc.WriteTo(b, from)
			}
		}
	}()
	return pc.LocalAddr().(*net.UDPAddr).AddrPort() //nolint:forcetypeassert // udp
}

func post(env *tooltest.Env, body string) tooltest.Response {
	return env.Do(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/snmp", bytes.NewBufferString(body)))
}

func TestSNMP(t *testing.T) {
	addr := agent(t)
	var redirected netip.AddrPort
	tool := &Tool{Redirect: func(a netip.AddrPort) netip.AddrPort { redirected = a; return addr }}
	env := tooltest.New(t, tool, tooltest.Options{Lookup: tooltest.StaticLookup("93.184.215.14"), Env: map[string]string{"SNMP_ENABLED": "true"}})

	var res Result
	post(env, `{"target":"router.example.com","community":"public","operation":"get","oids":["1.3.6.1.2.1.1.1.0","1.3.6.1.2.1.1.5.0","1.3.6.1.2.1.1.9.0"]}`).Must(t).Decode(t, &res)
	if redirected.String() != "93.184.215.14:161" || len(res.Vars) != 3 || res.Vars[0].Value != "Linux router 6.1" || res.Vars[0].Type != "OctetString" || res.Vars[2].Type != "NoSuchObject" {
		t.Errorf("get = %+v (dialed %s)", res, redirected)
	}

	post(env, `{"target":"router.example.com","community":"public","operation":"walk","oids":[".1.3.6.1.2.1"]}`).Must(t).Decode(t, &res)
	if len(res.Vars) != len(mib) || res.Truncated {
		t.Fatalf("walk = %+v", res)
	}
	byOID := map[string]Var{}
	for _, v := range res.Vars {
		byOID[v.OID] = v
	}
	if byOID[".1.3.6.1.2.1.2.2.1.6.1"].Value != "001122334455" || byOID[".1.3.6.1.2.1.2.2.1.6.1"].Type != "Hex-STRING" ||
		byOID[".1.3.6.1.2.1.31.1.1.1.6.1"].Value != "18000000000000000000" || byOID[".1.3.6.1.2.1.1.3.0"].Value != "123456" {
		t.Errorf("formatting = %+v", res.Vars)
	}

	post(env, `{"target":"router.example.com","community":"public","operation":"walk","oids":["1.3.6.1.2.1"],"maxRows":2}`).Must(t).Decode(t, &res)
	if len(res.Vars) != 2 || !res.Truncated {
		t.Errorf("bounded walk = %+v", res)
	}

	if r := post(env, `{"target":"router.example.com","community":"s3cret-xyz","operation":"get","oids":["1.3.6.1.2.1.1.1.0"]}`); r.OK || r.Error.Code != "PROVIDER_ERROR" || strings.Contains(r.Raw.Body.String(), "s3cret") {
		t.Errorf("wrong community = %s", r.Raw.Body.String())
	}
}

func TestSNMPValidation(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Env: map[string]string{"SNMP_ENABLED": "true"}})
	for body, field := range map[string]string{
		`{"target":"x.example","operation":"get","oids":["1.3.6"]}`:                                                                                                                                                        "community",
		`{"target":"x.example","community":"c","operation":"walk","oids":["1.3.6","1.3.7"]}`:                                                                                                                               "oids",
		`{"target":"x.example","community":"c","operation":"get","oids":["sysDescr"]}`:                                                                                                                                     "oids",
		`{"target":"x.example","community":"c","operation":"set","oids":["1.3.6"]}`:                                                                                                                                        "operation",
		`{"target":"x.example","community":"c","operation":"walk","oids":["1.3.6"],"maxRows":5000}`:                                                                                                                        "maxRows",
		`{"target":"x.example","version":"3","operation":"get","oids":["1.3.6"]}`:                                                                                                                                          "v3.user",
		`{"target":"x.example","version":"3","v3":{"user":"u","level":"authPriv","authProtocol":"SHA","authPassword":"short"},"operation":"get","oids":["1.3.6"]}`:                                                         "v3.authPassword",
		`{"target":"x.example","version":"3","v3":{"user":"u","level":"authPriv","authProtocol":"SHA","authPassword":"longenough","privProtocol":"ROT13","privPassword":"longenough"},"operation":"get","oids":["1.3.6"]}`: "v3.privProtocol",
	} {
		if r := post(env, body); r.OK || r.Error.Details["field"] != field {
			t.Errorf("%s: %+v", body, r.Error)
		}
	}
	if r := post(env, `{"target":"169.254.169.254","community":"c","operation":"get","oids":["1.3.6"]}`); r.OK || r.Error.Code != "TARGET_BLOCKED" {
		t.Errorf("metadata: %+v", r.Error)
	}
	if ok, reason := (&Tool{}).Enabled(tools.Deps{Config: tooltest.Config(t, nil)}); ok || reason != tools.ReasonDisabledByAdmin {
		t.Errorf("default: %v %s", ok, reason)
	}
}

func TestV3Client(t *testing.T) {
	req := Request{Version: "3", V3: &V3{User: "mon", Level: "authPriv", AuthProtocol: "SHA256", AuthPassword: "authpass1", PrivProtocol: "AES", PrivPassword: "privpass1"}}
	g := req.Client(netip.MustParseAddrPort("192.0.2.1:161"))
	sp, _ := g.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if g.Version != gosnmp.Version3 || g.MsgFlags != gosnmp.AuthPriv || sp.AuthenticationProtocol != gosnmp.SHA256 || sp.PrivacyProtocol != gosnmp.AES || sp.UserName != "mon" {
		t.Errorf("client = %+v %+v", g, sp)
	}
}

func TestNormalizeOID(t *testing.T) {
	for in, want := range map[string]string{"1.3.6.1": ".1.3.6.1", ".1.3.6.1.2.1.1.1.0": ".1.3.6.1.2.1.1.1.0", "3.1": "", "1": "", "1.3.x": "", "1..3": ""} {
		if got, _ := NormalizeOID(in); got != want {
			t.Errorf("%q = %q", in, got)
		}
	}
}
