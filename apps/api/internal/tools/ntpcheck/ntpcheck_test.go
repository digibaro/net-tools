package ntpcheck

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

const ntpEpochOffset = 2208988800

func ntpTime(t time.Time) uint64 {
	secs := uint64(t.Unix() + ntpEpochOffset)
	frac := uint64(t.Nanosecond()) << 32 / 1e9
	return secs<<32 | frac
}

// fakeServer answers like a stratum-2 server synced to 192.0.2.1, 50 ms ahead.
func fakeServer(t *testing.T) int {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0") //nolint:noctx // test listener
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pc.Close() })
	go func() {
		buf := make([]byte, 48)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil || n < 48 {
				return
			}
			now := time.Now().Add(50 * time.Millisecond)
			out := make([]byte, 48)
			out[0] = 0<<6 | 4<<3 | 4
			out[1], out[2], out[3] = 2, 6, 0xEC // stratum, poll, precision
			binary.BigEndian.PutUint32(out[4:], 0x00000100)
			binary.BigEndian.PutUint32(out[8:], 0x00000200)
			copy(out[12:16], []byte{192, 0, 2, 1})
			binary.BigEndian.PutUint64(out[16:], ntpTime(now.Add(-time.Minute)))
			copy(out[24:32], buf[40:48])
			binary.BigEndian.PutUint64(out[32:], ntpTime(now))
			binary.BigEndian.PutUint64(out[40:], ntpTime(now))
			_, _ = pc.WriteTo(out, addr)
		}
	}()
	return pc.LocalAddr().(*net.UDPAddr).Port
}

func TestNTP(t *testing.T) {
	port := fakeServer(t)
	env := tooltest.New(t, &Tool{Port: port}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/ntp?server=127.0.0.1").Must(t).Decode(t, &res)
	if res.Stratum != 2 || res.ReferenceID != "192.0.2.1" || !res.Valid || res.OffsetMs < 30 || res.OffsetMs > 70 || res.Leap != "none" {
		t.Errorf("res = %+v", res)
	}
}

func TestNTPSSRF(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Lookup: tooltest.StaticLookup("192.168.1.1")})
	for server, code := range map[string]string{
		"":                    "INVALID_INPUT",
		"127.0.0.1":           "TARGET_PRIVATE_DENIED",
		"ntp.lan":             "TARGET_PRIVATE_DENIED",
		"169.254.169.123":     "TARGET_PRIVATE_DENIED",
		"169.254.169.254":     "TARGET_BLOCKED",
		"http://pool.ntp.org": "INVALID_INPUT",
	} {
		env.Get("/api/v1/ntp?server="+server).ExpectError(t, code)
	}
}
