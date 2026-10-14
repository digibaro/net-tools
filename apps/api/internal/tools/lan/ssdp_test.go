package lan

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

type fakeSSDP struct {
	mu      sync.Mutex
	pending []struct {
		from string
		body string
	}
	searches int
}

func (f *fakeSSDP) WriteTo(p []byte, _ net.Addr) (int, error) {
	if !strings.HasPrefix(string(p), "M-SEARCH * HTTP/1.1") {
		return 0, errors.New("not an M-SEARCH")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searches++
	reply := func(from, st, loc string) {
		f.pending = append(f.pending, struct{ from, body string }{from, "HTTP/1.1 200 OK\r\nCACHE-CONTROL: max-age=1800\r\nLOCATION: " + loc + "\r\nSERVER: Linux UPnP/1.0 MiniUPnPd/2.3\r\nST: " + st + "\r\nUSN: uuid:abc::" + st + "\r\n\r\n"})
	}
	reply("192.168.1.1", "upnp:rootdevice", "http://192.168.1.1:5000/rootDesc.xml")
	reply("192.168.1.1", "urn:schemas-upnp-org:device:InternetGatewayDevice:1", "http://192.168.1.1:5000/rootDesc.xml")
	reply("192.168.1.30", "urn:dial-multiscreen-org:service:dial:1", "http://10.20.30.40/desc.xml") // hosted elsewhere
	return len(p), nil
}

func (f *fakeSSDP) ReadFrom(p []byte) (int, net.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.pending) == 0 {
		return 0, nil, timeoutErr{}
	}
	r := f.pending[0]
	f.pending = f.pending[1:]
	return copy(p, r.body), &net.UDPAddr{IP: net.ParseIP(r.from), Port: 1900}, nil
}

func (f *fakeSSDP) SetReadDeadline(time.Time) error { return nil }
func (f *fakeSSDP) Close() error                    { return nil }

func TestSSDP(t *testing.T) {
	fetched := map[string]bool{}
	var mu sync.Mutex
	fetch := func(_ context.Context, u string) ([]byte, error) {
		mu.Lock()
		fetched[u] = true
		mu.Unlock()
		return []byte(`<?xml version="1.0"?><root xmlns="urn:schemas-upnp-org:device-1-0"><device><deviceType>urn:schemas-upnp-org:device:InternetGatewayDevice:1</deviceType><friendlyName>Home Router</friendlyName><manufacturer>ACME</manufacturer><modelName>RT</modelName><modelNumber>9000</modelNumber></device></root>`), nil
	}
	tool := &SSDPTool{Listen: func(context.Context) (PacketConn, error) { return &fakeSSDP{}, nil }, Fetch: fetch}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	body := env.Get("/api/v1/lan/ssdp/stream?seconds=1").Raw.Body.String()
	i := strings.Index(body, "event: done\ndata: ")
	if i < 0 {
		t.Fatalf("body = %s", body)
	}
	var d struct {
		Summary SSDPSummary `json:"summary"`
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(body[i+len("event: done\ndata: "):])), &d)
	devs := d.Summary.Devices
	if len(devs) != 2 || devs[0].FriendlyName != "Home Router" || devs[0].Model != "RT 9000" || len(devs[0].Types) != 2 || devs[1].Error == "" {
		t.Fatalf("devices = %+v", devs)
	}
	if fetched["http://10.20.30.40/desc.xml"] {
		t.Error("fetched a description hosted on another address")
	}
}

func TestParseSSDPResponse(t *testing.T) {
	h, err := ParseSSDPResponse([]byte("HTTP/1.1 200 OK\r\nLocation: http://x/\r\nst: upnp:rootdevice\r\n\r\n"))
	if err != nil || h["LOCATION"] != "http://x/" || h["ST"] != "upnp:rootdevice" {
		t.Errorf("h = %v %v", h, err)
	}
}
