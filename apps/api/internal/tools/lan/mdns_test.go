package lan

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeMulticast answers mDNS questions like a printer and a NAS would.
type fakeMulticast struct {
	mu      sync.Mutex
	pending [][]byte
	closed  bool
}

func rrs(lines ...string) []dns.RR {
	var out []dns.RR
	for _, l := range lines {
		r, err := dns.NewRR(l)
		if err != nil {
			panic(err)
		}
		out = append(out, r)
	}
	return out
}

func (f *fakeMulticast) WriteTo(p []byte, _ net.Addr) (int, error) {
	q := new(dns.Msg)
	if err := q.Unpack(p); err != nil {
		return 0, err
	}
	for _, question := range q.Question {
		if question.Qclass&(1<<15) == 0 {
			panic("QU bit not set")
		}
		resp := new(dns.Msg)
		resp.Response = true
		switch question.Name {
		case servicesName:
			resp.Answer = rrs(servicesName+" 4500 IN PTR _ipp._tcp.local.", servicesName+" 4500 IN PTR _smb._tcp.local.")
		case "_ipp._tcp.local.":
			resp.Answer = rrs(`_ipp._tcp.local. 4500 IN PTR Office\ Printer._ipp._tcp.local.`)
			resp.Extra = rrs(`Office\ Printer._ipp._tcp.local. 120 IN SRV 0 0 631 printer.local.`, `Office\ Printer._ipp._tcp.local. 4500 IN TXT "ty=LaserJet" "rp=ipp/print"`, "printer.local. 120 IN A 192.168.1.50")
		case "_smb._tcp.local.":
			resp.Answer = rrs("_smb._tcp.local. 4500 IN PTR nas._smb._tcp.local.")
		case "nas._smb._tcp.local.":
			resp.Answer = rrs("nas._smb._tcp.local. 120 IN SRV 0 0 445 nas.local.", "nas.local. 120 IN A 192.168.1.10")
		default:
			continue
		}
		b, _ := resp.Pack()
		f.mu.Lock()
		f.pending = append(f.pending, b)
		f.mu.Unlock()
	}
	return len(p), nil
}

func (f *fakeMulticast) ReadFrom(p []byte) (int, net.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.pending) == 0 {
		return 0, nil, timeoutErr{}
	}
	b := f.pending[0]
	f.pending = f.pending[1:]
	return copy(p, b), &net.UDPAddr{IP: net.ParseIP("192.168.1.50"), Port: 5353}, nil
}

func (f *fakeMulticast) SetReadDeadline(time.Time) error { return nil }
func (f *fakeMulticast) Close() error                    { f.closed = true; return nil }

func TestMDNS(t *testing.T) {
	fake := &fakeMulticast{}
	tool := &MDNSTool{Listen: func(context.Context) (PacketConn, error) { return fake, nil }}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	body := env.Get("/api/v1/lan/mdns/stream?seconds=2").Raw.Body.String()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("body = %s", body)
	}
	var d struct {
		Summary MDNSSummary `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	s := d.Summary
	if len(s.Types) != 2 || len(s.Services) != 2 || !fake.closed {
		t.Fatalf("summary = %+v", s)
	}
	printer := s.Services[0]
	if printer.Type != "_ipp._tcp.local." || printer.Port != 631 || printer.Host != "printer.local." || printer.Addresses[0] != "192.168.1.50" || printer.TXT[0] != "ty=LaserJet" {
		t.Errorf("printer = %+v", printer)
	}
	if s.Services[1].Port != 445 || s.Services[1].Addresses[0] != "192.168.1.10" {
		t.Errorf("nas (resolved by follow-up query) = %+v", s.Services[1])
	}
	if b := env.Get("/api/v1/lan/mdns/stream?service=http").Raw.Body.String(); !strings.Contains(b, "INVALID_INPUT") {
		t.Error(b)
	}
	if b := env.Get("/api/v1/lan/mdns/stream?seconds=60").Raw.Body.String(); !strings.Contains(b, "INVALID_INPUT") {
		t.Error(b)
	}
}
