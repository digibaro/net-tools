package tcping

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

func TestTCPingOpenAndClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx // test listener
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	env := tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true, Env: map[string]string{"PING_MIN_INTERVAL_MS": "50"}})
	body := env.Get("/api/v1/tcping/stream?target=127.0.0.1&count=3&interval=50&port=" + itoa(port)).Raw.Body.String()
	if strings.Count(body, "event: reply") != 3 || !strings.Contains(body, `"lossPct":0`) {
		t.Fatalf("open port: %s", body)
	}
	ln.Close()
	body = env.Get("/api/v1/tcping/stream?target=127.0.0.1&count=2&interval=50&port=" + itoa(port)).Raw.Body.String()
	if strings.Count(body, "event: timeout") != 2 || !strings.Contains(body, "connection refused") || !strings.Contains(body, `"lossPct":100`) {
		t.Errorf("closed port: %s", body)
	}
}

func TestTCPingValidationAndSSRF(t *testing.T) {
	env := tooltest.New(t, &Tool{}, tooltest.Options{Lookup: tooltest.StaticLookup("192.168.1.1")})
	for q, code := range map[string]string{
		"target=8.8.8.8&port=0":                          "INVALID_INPUT",
		"target=8.8.8.8&port=70000":                      "INVALID_INPUT",
		"target=8.8.8.8&count=100":                       "INVALID_INPUT",
		"target=127.0.0.1&port=22":                       "TARGET_PRIVATE_DENIED",
		"target=router.example":                          "TARGET_PRIVATE_DENIED",
		"target=169.254.169.254":                         "TARGET_BLOCKED",
		"target=" + url.QueryEscape("[::ffff:10.0.0.1]"): "TARGET_PRIVATE_DENIED",
	} {
		body := env.Get("/api/v1/tcping/stream?" + q).Raw.Body.String()
		if !strings.Contains(body, `"code":"`+code+`"`) {
			t.Errorf("%s: %s", q, body)
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
