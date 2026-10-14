package sshaudit

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"net"
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/digibaro/net-tools/apps/api/internal/tools/tooltest"
)

func nameList(names ...string) []byte {
	s := strings.Join(names, ",")
	b := make([]byte, 4+len(s))
	binary.BigEndian.PutUint32(b, uint32(len(s))) //nolint:gosec // test data
	copy(b[4:], s)
	return b
}

func TestParseKexInitAndAnalyze(t *testing.T) {
	p := append([]byte{20}, make([]byte, 16)...)
	for _, l := range [][]string{
		{"curve25519-sha256", "diffie-hellman-group1-sha1"}, {"ssh-ed25519", "ssh-rsa"},
		{"chacha20-poly1305@openssh.com", "aes128-ctr"}, {"aes128-ctr", "3des-cbc"},
		{"hmac-sha2-256-etm@openssh.com"}, {"hmac-sha2-256-etm@openssh.com", "hmac-md5"},
		{"none"}, {"none", "zlib@openssh.com"}, {}, {},
	} {
		p = append(p, nameList(l...)...)
	}
	p = append(p, 0, 0, 0, 0, 0)
	k, err := ParseKexInit(p)
	if err != nil || k.Kex[1] != "diffie-hellman-group1-sha1" || !slices.Equal(k.Ciphers, []string{"chacha20-poly1305@openssh.com", "aes128-ctr", "3des-cbc"}) || len(k.Compression) != 2 {
		t.Fatalf("k = %+v %v", k, err)
	}
	f, terrapin := Analyze(k)
	if !terrapin {
		t.Error("terrapin not detected")
	}
	has := func(alg, sev string) bool {
		return slices.ContainsFunc(f, func(x Finding) bool { return x.Algorithm == alg && x.Severity == sev })
	}
	if !has("diffie-hellman-group1-sha1", "fail") || !has("ssh-rsa", "warn") || !has("3des-cbc", "fail") || !has("hmac-md5", "fail") {
		t.Errorf("findings = %+v", f)
	}
	k.Kex = append(k.Kex, "kex-strict-s-v00@openssh.com", "mlkem768x25519-sha256")
	if _, terrapin := Analyze(k); terrapin {
		t.Error("strict kex should mitigate Terrapin")
	}
	if _, err := ParseKexInit([]byte{21, 1, 2}); err == nil {
		t.Error("non-KEXINIT accepted")
	}
}

func TestAuditAgainstGoServer(t *testing.T) {
	_, edKey, _ := ed25519.GenerateKey(rand.Reader)
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	cfg := &ssh.ServerConfig{PasswordCallback: func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) { return nil, ssh.ErrNoAuth }}
	for _, k := range []any{edKey, rsaKey} {
		s, err := ssh.NewSignerFromKey(k)
		if err != nil {
			t.Fatal(err)
		}
		cfg.AddHostKey(s)
	}
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
			go func() {
				defer c.Close()
				_, _, _, _ = ssh.NewServerConn(c, cfg)
			}()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	var out Result
	tooltest.New(t, &Tool{}, tooltest.Options{AllowPrivate: true}).Get("/api/v1/ssh/audit?host=127.0.0.1&port="+strconv.Itoa(port)).Must(t).Decode(t, &out)
	if !strings.HasPrefix(out.Banner, "SSH-2.0-Go") || len(out.Algos.Kex) == 0 || len(out.HostKeys) != 2 || !strings.HasPrefix(out.HostKeys[0].Fingerprint, "SHA256:") {
		t.Errorf("out = %+v", out)
	}
	for _, hk := range out.HostKeys {
		if hk.Type == "ssh-rsa" && hk.Bits != 2048 {
			t.Errorf("rsa bits = %d", hk.Bits)
		}
	}
	tooltest.New(t, &Tool{}, tooltest.Options{}).Get("/api/v1/ssh/audit?host=10.0.0.22").ExpectError(t, "TARGET_PRIVATE_DENIED")
}
