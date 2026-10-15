package portcheck

import (
	"net"
	"slices"
	"strconv"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestParsePorts(t *testing.T) {
	got, err := ParsePorts("22, 80,443,8000-8002,80")
	if err != nil || !slices.Equal(got, []int{22, 80, 443, 8000, 8001, 8002}) {
		t.Errorf("%v %v", got, err)
	}
	for _, bad := range []string{"", "0", "70000", "a", "10-5", "1-20", "1,2,3,4,5,6,7,8,9,10,11"} {
		if _, err := ParsePorts(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestPortCheck(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test listener
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	open := ln.Addr().(*net.TCPAddr).Port
	closedLn, _ := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test listener
	closed := closedLn.Addr().(*net.TCPAddr).Port
	closedLn.Close()
	udp, _ := net.ListenPacket("udp", "127.0.0.1:0") //nolint:noctx // test listener
	defer udp.Close()
	go func() {
		buf := make([]byte, 64)
		for {
			_, a, err := udp.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = udp.WriteTo([]byte("pong"), a)
		}
	}()
	udpPort := udp.LocalAddr().(*net.UDPAddr).Port

	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true})
	var res Result
	env.Get("/api/v1/port/check?target=127.0.0.1&ports="+strconv.Itoa(open)+","+strconv.Itoa(closed)).Must(t).Decode(t, &res)
	if res.Rows[0].State != "open" || res.Rows[1].State != "closed" {
		t.Errorf("tcp rows = %+v", res.Rows)
	}
	env.Get("/api/v1/port/check?proto=udp&target=127.0.0.1&ports="+strconv.Itoa(udpPort)+","+strconv.Itoa(closed)).Must(t).Decode(t, &res)
	if res.Rows[0].State != "open" || res.Rows[1].State != "closed" {
		t.Errorf("udp rows = %+v", res.Rows)
	}
}

func TestPortCheckSSRF(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Lookup: tooltest.StaticLookup("10.10.10.10")})
	for q, code := range map[string]string{
		"target=8.8.8.8":                        "INVALID_INPUT",
		"target=8.8.8.8&ports=1-11":             "INVALID_INPUT",
		"target=8.8.8.8&ports=22&proto=sctp":    "INVALID_INPUT",
		"target=127.0.0.1&ports=22":             "TARGET_PRIVATE_DENIED",
		"target=db.internal.example&ports=5432": "TARGET_PRIVATE_DENIED",
		"target=169.254.169.254&ports=80":       "TARGET_BLOCKED",
		"target=[fd00::1]&ports=22":             "TARGET_PRIVATE_DENIED",
		"target=0.0.0.0&ports=22":               "TARGET_BLOCKED",
	} {
		env.Get("/api/v1/port/check?"+q).ExpectError(t, code)
	}
}
