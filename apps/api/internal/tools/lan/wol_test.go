package lan

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/tools"
	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func TestMagicPacket(t *testing.T) {
	mac, _ := net.ParseMAC("00:11:22:33:44:55")
	p := MagicPacket(mac)
	if len(p) != 102 || !bytes.Equal(p[:6], bytes.Repeat([]byte{0xFF}, 6)) || !bytes.Equal(p[96:], mac) {
		t.Errorf("packet = %x", p)
	}
}

func TestWOL(t *testing.T) {
	var sentTo netip.AddrPort
	var sent []byte
	send := func(_ context.Context, dst netip.AddrPort, payload []byte) error {
		sentTo, sent = dst, payload
		return nil
	}
	env := tooltest.New(t, &WOLTool{Send: send}, tooltest.Options{Env: map[string]string{"NETTOOLS_LAN": "true"}})
	post := func(body string) tooltest.Response {
		return env.Do(httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/lan/wol", strings.NewReader(body)))
	}
	var res WOLResult
	post(`{"mac":"00-11-22-33-44-55","broadcast":"192.168.1.255"}`).Must(t).Decode(t, &res)
	if res.Destination != "192.168.1.255:9" || sentTo.String() != "192.168.1.255:9" || len(sent) != 102 || res.MAC != "00:11:22:33:44:55" {
		t.Errorf("res = %+v sent to %s", res, sentTo)
	}
	post(`{"mac":"00:11:22:33:44:55"}`).Must(t).Decode(t, &res)
	if res.Destination != "255.255.255.255:9" {
		t.Errorf("default = %+v", res)
	}
	for body, field := range map[string]string{
		`{"mac":"nope"}`: "mac", `{"mac":"00:11:22:33:44:55","broadcast":"8.8.8.255"}`: "broadcast",
		`{"mac":"00:11:22:33:44:55","port":53}`: "port", `{"mac":"00:11:22:33:44:55:66:77"}`: "mac",
	} {
		r := post(body)
		if r.OK || r.Error.Code != "INVALID_INPUT" || r.Error.Details["field"] != field {
			t.Errorf("%s: %+v", body, r.Error)
		}
	}
	if ok, reason := (&WOLTool{}).Enabled(tools.Deps{Config: tooltest.Config(t, nil)}); ok || reason != tools.ReasonRequiresLANMode {
		t.Errorf("without LAN mode: %v %s", ok, reason)
	}
}

// The real sender on loopback: catches socket-option mistakes the fake sender hides.
func TestUDPBroadcastSocket(t *testing.T) {
	pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Skip(err)
	}
	defer pc.Close()
	dst := pc.LocalAddr().(*net.UDPAddr).AddrPort() //nolint:forcetypeassert // udp
	mac, _ := net.ParseMAC("02:00:5e:00:00:01")
	if err := udpBroadcast(context.Background(), dst, MagicPacket(mac)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 200)
	_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
	if n, _, err := pc.ReadFrom(buf); err != nil || n != 102 {
		t.Errorf("received %d bytes, %v", n, err)
	}
}
