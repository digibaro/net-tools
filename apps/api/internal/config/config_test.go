package config

import (
	"errors"
	"io/fs"
	"log/slog"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) LookupFunc {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func files(m map[string]string) ReadFileFunc {
	return func(p string) ([]byte, error) {
		if v, ok := m[p]; ok {
			return []byte(v), nil
		}
		return nil, fs.ErrNotExist
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadFrom(env(nil), files(nil))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Profile != ProfilePublic {
		t.Errorf("Profile = %q, want public", cfg.Profile)
	}
	if cfg.Listen != "0.0.0.0:8081" {
		t.Errorf("Listen = %q", cfg.Listen)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v", cfg.LogLevel)
	}
	if cfg.Features.AllowPrivate || cfg.Features.Portscan || cfg.Features.ServerInfo {
		t.Errorf("public profile must deny private targets and disable portscan and server info")
	}
	want := Limits{
		RateLimitRPM: 60, RateLimitHeavyRPM: 10, ConcurrencyGlobal: 32, ConcurrencyPerIP: 3,
		PingMaxCount: 20, PingMinInterval: 200 * time.Millisecond, TracerouteMaxHops: 30,
		MTRMaxCycles: 20, PortscanMaxPorts: 1024, PortscanRatePPS: 200, IperfMaxSeconds: 15,
		HTTPMaxBodyBytes: 1 << 20, ToolTimeoutDefault: 15 * time.Second,
		StreamTimeoutMax: 120 * time.Second, BulkMaxTargets: 50,
	}
	if cfg.Limits != want {
		t.Errorf("Limits = %+v, want %+v", cfg.Limits, want)
	}
	if got := cfg.Features.TrustedProxies; len(got) != 1 || got[0] != netip.MustParsePrefix("172.16.0.0/12") {
		t.Errorf("TrustedProxies = %v", got)
	}
}

func TestLoadProfiles(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		allowPrivate bool
		portscan     bool
	}{
		{"public", map[string]string{"NETTOOLS_PROFILE": "public"}, false, false},
		{"internal", map[string]string{"NETTOOLS_PROFILE": "internal"}, true, true},
		{"homelab uppercase", map[string]string{"NETTOOLS_PROFILE": "HOMELAB"}, true, true},
		{"override wins", map[string]string{"NETTOOLS_PROFILE": "homelab", "ALLOW_PRIVATE_TARGETS": "false", "PORTSCAN_ENABLED": "false"}, false, false},
		{"empty value uses profile default", map[string]string{"NETTOOLS_PROFILE": "internal", "ALLOW_PRIVATE_TARGETS": ""}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFrom(env(tt.env), files(nil))
			if err != nil {
				t.Fatalf("LoadFrom: %v", err)
			}
			if cfg.Features.AllowPrivate != tt.allowPrivate || cfg.Features.Portscan != tt.portscan {
				t.Errorf("AllowPrivate=%v Portscan=%v, want %v %v",
					cfg.Features.AllowPrivate, cfg.Features.Portscan, tt.allowPrivate, tt.portscan)
			}
		})
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := LoadFrom(env(map[string]string{
		"API_LISTEN":           "127.0.0.1:9000",
		"LOG_LEVEL":            "debug",
		"PING_MAX_COUNT":       "5",
		"TOOL_TIMEOUT_DEFAULT": "3s",
		"DENY_CIDRS":           "10.1.0.0/16, 192.0.2.7 ,2001:db8::/32",
	}), files(nil))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Listen != "127.0.0.1:9000" || cfg.LogLevel != slog.LevelDebug {
		t.Errorf("Listen=%q LogLevel=%v", cfg.Listen, cfg.LogLevel)
	}
	if cfg.Limits.PingMaxCount != 5 || cfg.Limits.ToolTimeoutDefault != 3*time.Second {
		t.Errorf("Limits = %+v", cfg.Limits)
	}
	want := []netip.Prefix{
		netip.MustParsePrefix("10.1.0.0/16"),
		netip.MustParsePrefix("192.0.2.7/32"),
		netip.MustParsePrefix("2001:db8::/32"),
	}
	if len(cfg.Features.DenyCIDRs) != len(want) {
		t.Fatalf("DenyCIDRs = %v", cfg.Features.DenyCIDRs)
	}
	for i := range want {
		if cfg.Features.DenyCIDRs[i] != want[i] {
			t.Errorf("DenyCIDRs[%d] = %v, want %v", i, cfg.Features.DenyCIDRs[i], want[i])
		}
	}
}

func TestLoadInvalid(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"profile", map[string]string{"NETTOOLS_PROFILE": "open"}, "NETTOOLS_PROFILE"},
		{"int", map[string]string{"PING_MAX_COUNT": "lots"}, "PING_MAX_COUNT"},
		{"negative int", map[string]string{"CONCURRENCY_GLOBAL": "-1"}, "CONCURRENCY_GLOBAL"},
		{"bool", map[string]string{"PORTSCAN_ENABLED": "maybe"}, "PORTSCAN_ENABLED"},
		{"duration", map[string]string{"TOOL_TIMEOUT_DEFAULT": "15"}, "TOOL_TIMEOUT_DEFAULT"},
		{"level", map[string]string{"LOG_LEVEL": "loud"}, "LOG_LEVEL"},
		{"cidr", map[string]string{"ALLOW_CIDRS": "10.0.0.0/33"}, "ALLOW_CIDRS"},
		{"secret file", map[string]string{"IPINFO_TOKEN_FILE": "/run/secrets/missing"}, "IPINFO_TOKEN_FILE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFrom(env(tt.env), files(nil))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want mention of %s", err, tt.want)
			}
		})
	}
}

func TestLoadReportsAllErrors(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"PING_MAX_COUNT": "x", "LOG_LEVEL": "y"}), files(nil))
	if err == nil || !strings.Contains(err.Error(), "PING_MAX_COUNT") || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("err = %v, want both variables reported", err)
	}
}

func TestSecrets(t *testing.T) {
	cfg, err := LoadFrom(
		env(map[string]string{
			"IPINFO_TOKEN":        " abc ",
			"ABUSEIPDB_KEY_FILE":  "/run/secrets/abuse",
			"GLOBALPING_TOKEN":    "",
			"VIRUSTOTAL_KEY_FILE": "",
		}),
		files(map[string]string{"/run/secrets/abuse": "s3cret\n"}),
	)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got := cfg.Features.ProviderSecrets["IPINFO_TOKEN"]; got != "abc" {
		t.Errorf("IPINFO_TOKEN = %q", got)
	}
	if got := cfg.Features.ProviderSecrets["ABUSEIPDB_KEY"]; got != "s3cret" {
		t.Errorf("ABUSEIPDB_KEY = %q", got)
	}
	if cfg.HasSecret("GLOBALPING_TOKEN") || cfg.HasSecret("VIRUSTOTAL_KEY") {
		t.Errorf("empty secrets must be treated as missing")
	}
}

func TestSecretFileErrorWrapped(t *testing.T) {
	_, err := LoadFrom(env(map[string]string{"IPINFO_TOKEN_FILE": "/nope"}), files(nil))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("err = %v, want wrapped fs.ErrNotExist", err)
	}
}
