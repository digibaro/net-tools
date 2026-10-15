package email

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/tools"
	"github.com/digibaro/net-tools-docker/apps/api/internal/tools/tooltest"
)

// fakeMTA serves one SMTP session over a pipe. open decides whether a recipient is accepted.
func fakeMTA(t *testing.T, open func(rcpt string) bool, log *[]string, mu *sync.Mutex) func(context.Context, string) (net.Conn, error) {
	t.Helper()
	return func(context.Context, string) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			w := bufio.NewWriter(server)
			say := func(s string) { _, _ = w.WriteString(s + "\r\n"); _ = w.Flush() }
			say("220 mx.example.com ESMTP")
			r := bufio.NewReader(server)
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return
				}
				line = strings.TrimRight(line, "\r\n")
				mu.Lock()
				*log = append(*log, line)
				mu.Unlock()
				switch up := strings.ToUpper(line); {
				case strings.HasPrefix(up, "EHLO"):
					say("250-mx.example.com\r\n250-PIPELINING\r\n250 SIZE 10240000")
				case strings.HasPrefix(up, "MAIL FROM:"):
					say("250 2.1.0 Ok")
				case strings.HasPrefix(up, "RCPT TO:"):
					if open(line) {
						say("250 2.1.5 Ok")
					} else {
						say("554 5.7.1 <relay-test@example.net>: Relay access denied")
					}
				case up == "RSET":
					say("250 2.0.0 Ok")
				case up == "QUIT":
					say("221 2.0.0 Bye")
					return
				case up == "DATA":
					t.Error("client sent DATA")
					return
				}
			}
		}()
		return client, nil
	}
}

func TestRelayClosed(t *testing.T) {
	var log []string
	var mu sync.Mutex
	tool := &RelayTool{Dial: fakeMTA(t, func(string) bool { return false }, &log, &mu)}
	env := tooltest.New(t, tool, tooltest.Options{Lookup: tooltest.StaticLookup("93.184.215.14"), Env: map[string]string{"OPEN_RELAY_TEST_ENABLED": "true"}})
	var res RelayResult
	env.Get("/api/v1/email/relay?host=mx.example.com&domain=Example.COM").Must(t).Decode(t, &res)
	if res.Open || len(res.Cases) != 5 || res.Error != "" || res.Address != "93.184.215.14:25" || !strings.Contains(res.Cases[0].RcptReply, "554 5.7.1") || res.Cases[4].MailFrom != "postmaster@example.com" {
		t.Errorf("res = %+v", res)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Count(strings.Join(log, "\n"), "RSET") != 5 || log[len(log)-1] != "QUIT" {
		t.Errorf("session = %v", log)
	}
}

func TestRelayOpen(t *testing.T) {
	var log []string
	var mu sync.Mutex
	tool := &RelayTool{Dial: fakeMTA(t, func(rcpt string) bool { return strings.Contains(rcpt, "%") }, &log, &mu)}
	env := tooltest.New(t, tool, tooltest.Options{Lookup: tooltest.StaticLookup("93.184.215.14"), Env: map[string]string{"OPEN_RELAY_TEST_ENABLED": "true"}})
	var res RelayResult
	env.Get("/api/v1/email/relay?host=mx.example.com").Must(t).Decode(t, &res)
	if !res.Open || len(res.Cases) != 4 || !res.Cases[2].Accepted || res.Cases[0].Accepted || res.Cases[2].RcptTo != "relay-test%example.net@[93.184.215.14]" {
		t.Errorf("res = %+v", res)
	}
}

func TestRelayValidation(t *testing.T) {
	env := tooltest.New(t, &RelayTool{}, tooltest.Options{Env: map[string]string{"OPEN_RELAY_TEST_ENABLED": "true"}})
	env.Get("/api/v1/email/relay?host=mx.example.com&port=80").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/email/relay?host=mx.example.com&domain=not_a_domain").ExpectError(t, "INVALID_INPUT")
	env.Get("/api/v1/email/relay?host=10.0.0.1").ExpectError(t, "TARGET_PRIVATE_DENIED")
	if ok, reason := (&RelayTool{}).Enabled(tools.Deps{Config: tooltest.Config(t, nil)}); ok || reason != tools.ReasonDisabledByAdmin {
		t.Errorf("default: %v %s", ok, reason)
	}
}
