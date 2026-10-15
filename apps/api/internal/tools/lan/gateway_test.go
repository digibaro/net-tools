package lan

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestParseNATPMP(t *testing.T) {
	ip, epoch, err := ParseNATPMP([]byte{0, 128, 0, 0, 0, 0, 0, 42, 203, 0, 113, 7})
	if err != nil || ip.String() != "203.0.113.7" || epoch != 42 {
		t.Errorf("%v %d %v", ip, epoch, err)
	}
	if _, _, err := ParseNATPMP([]byte{2, 0x80, 0, 1}); !errors.Is(err, errPCPOnly) {
		t.Error(err)
	}
	if _, _, err := ParseNATPMP([]byte{0, 128, 0, 3, 0, 0, 0, 0, 0, 0, 0, 0}); err == nil {
		t.Error("result code accepted")
	}
}

func TestOnGateway(t *testing.T) {
	gw := netip.MustParseAddr("192.168.1.1")
	if u, ok := onGateway("http://192.168.1.1:5000/rootDesc.xml", "/ctl/IPConn", gw); !ok || u != "http://192.168.1.1:5000/ctl/IPConn" {
		t.Error(u, ok)
	}
	for _, ref := range []string{"http://203.0.113.9/ctl", "https://192.168.1.1/ctl", "file:///etc/passwd"} {
		if _, ok := onGateway("http://192.168.1.1:5000/", ref, gw); ok {
			t.Errorf("%s allowed", ref)
		}
	}
}

// fakeIGD answers the IGD search from the gateway and from an impostor.
type fakeIGD struct {
	mu      sync.Mutex
	pending []string
	from    []string
}

func (f *fakeIGD) WriteTo(p []byte, _ net.Addr) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = append(f.pending, "HTTP/1.1 200 OK\r\nLOCATION: http://192.168.1.77/evil.xml\r\nST: x\r\n\r\n", "HTTP/1.1 200 OK\r\nLOCATION: http://192.168.1.1:5000/rootDesc.xml\r\nST: x\r\n\r\n")
	f.from = append(f.from, "192.168.1.77", "192.168.1.1")
	return len(p), nil
}

func (f *fakeIGD) ReadFrom(p []byte) (int, net.Addr, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.pending) == 0 {
		return 0, nil, timeoutErr{}
	}
	b, from := f.pending[0], f.from[0]
	f.pending, f.from = f.pending[1:], f.from[1:]
	return copy(p, b), &net.UDPAddr{IP: net.ParseIP(from), Port: 1900}, nil
}

func (f *fakeIGD) SetReadDeadline(time.Time) error { return nil }
func (f *fakeIGD) Close() error                    { return nil }

const rootDesc = `<?xml version="1.0"?><root xmlns="urn:schemas-upnp-org:device-1-0"><device><deviceType>urn:schemas-upnp-org:device:InternetGatewayDevice:1</deviceType><friendlyName>Home Router</friendlyName><modelName>RT-9000</modelName>
<deviceList><device><deviceType>urn:schemas-upnp-org:device:WANDevice:1</deviceType><deviceList><device><deviceType>urn:schemas-upnp-org:device:WANConnectionDevice:1</deviceType>
<serviceList><service><serviceType>urn:schemas-upnp-org:service:WANIPConnection:1</serviceType><controlURL>/ctl/IPConn</controlURL></service></serviceList></device></deviceList></device></deviceList></device></root>`

const soapResp = `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:GetExternalIPAddressResponse xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:1"><NewExternalIPAddress>100.64.12.34</NewExternalIPAddress></u:GetExternalIPAddressResponse></s:Body></s:Envelope>`

func TestGateway(t *testing.T) {
	var calls []string
	var mu sync.Mutex
	tool := &GatewayTool{
		ReadFile: func(p string) ([]byte, error) {
			if p == "/proc/net/arp" {
				return []byte("IP address HW type Flags HW address Mask Device\n192.168.1.1 0x1 0x2 aa:bb:cc:dd:ee:01 * eth0\n"), nil
			}
			return []byte(procRoute), nil
		},
		Addrs: func(string) []string { return []string{"192.168.1.20/24"} },
		NATPMP: func(context.Context, netip.Addr) NATPMPResult {
			return NATPMPResult{Error: "no NAT-PMP/PCP answer from the gateway"}
		},
		Listen: func(context.Context) (PacketConn, error) { return &fakeIGD{}, nil },
		HTTP: func(_ context.Context, method, u, action string, body []byte) ([]byte, error) {
			mu.Lock()
			calls = append(calls, method+" "+u)
			mu.Unlock()
			if method == "GET" {
				return []byte(rootDesc), nil
			}
			if !strings.Contains(action, "WANIPConnection:1#GetExternalIPAddress") || !strings.Contains(string(body), "GetExternalIPAddress") {
				return nil, errors.New("bad SOAP call")
			}
			return []byte(soapResp), nil
		},
	}
	env := tooltest.New(t, tool, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	var res GatewayResult
	env.Get("/api/v1/lan/gateway").Must(t).Decode(t, &res)
	if res.Interface != "eth0" || res.Gateway != "192.168.1.1" || res.GatewayMAC != "aa:bb:cc:dd:ee:01" || res.Addresses[0] != "192.168.1.20/24" {
		t.Errorf("basics = %+v", res)
	}
	if !res.UPnP.Supported || res.UPnP.ExternalIP != "100.64.12.34" || res.UPnP.FriendlyName != "Home Router" || res.ExternalIP != "100.64.12.34" || !res.DoubleNAT {
		t.Errorf("upnp = %+v / %+v", res.UPnP, res)
	}
	if len(calls) != 2 || calls[0] != "GET http://192.168.1.1:5000/rootDesc.xml" || calls[1] != "POST http://192.168.1.1:5000/ctl/IPConn" {
		t.Errorf("calls = %v (must ignore the impostor at .77)", calls)
	}
}

func TestNATPMPOverUDP(t *testing.T) {
	pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Skip(err)
	}
	defer pc.Close()
	go func() {
		buf := make([]byte, 64)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			switch {
			case n == 2 && buf[0] == 0 && buf[1] == 0: // NAT-PMP external address
				_, _ = pc.WriteTo([]byte{0, 128, 0, 0, 0, 0, 1, 0, 198, 51, 100, 4}, from)
			case n == 24 && buf[0] == 2 && buf[1] == 0: // PCP ANNOUNCE
				_, _ = pc.WriteTo(append([]byte{2, 0x80, 0, 0}, make([]byte, 20)...), from)
			}
		}
	}()
	port := pc.LocalAddr().(*net.UDPAddr).Port
	r := natPMPAt(context.Background(), netip.MustParseAddr("127.0.0.1"), port)
	if !r.Supported || r.ExternalIP != "198.51.100.4" || r.Epoch != 256 || !r.PCP || r.Error != "" {
		t.Errorf("r = %+v", r)
	}
}
