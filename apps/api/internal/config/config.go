// Package config loads NetTools API configuration from environment variables.
//
// Defaults follow docs/SECURITY.md; NETTOOLS_PROFILE selects profile defaults and individual
// variables override them. Secrets may be given as <NAME>_FILE pointing at a file (Docker secrets).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

// Profile is a deployment profile (docs/SECURITY.md §1).
type Profile string

const (
	ProfilePublic   Profile = "public"
	ProfileInternal Profile = "internal"
	ProfileHomelab  Profile = "homelab"
)

// Limits are per-tool and global bounds enforced by server tools.
type Limits struct {
	RateLimitRPM       int
	RateLimitHeavyRPM  int
	ConcurrencyGlobal  int
	ConcurrencyPerIP   int
	PingMaxCount       int
	PingMinInterval    time.Duration
	TracerouteMaxHops  int
	MTRMaxCycles       int
	PortscanMaxPorts   int
	PortscanRatePPS    int
	IperfMaxSeconds    int
	HTTPMaxBodyBytes   int64
	ToolTimeoutDefault time.Duration
	StreamTimeoutMax   time.Duration
	BulkMaxTargets     int
}

// Features are admin feature flags.
type Features struct {
	Portscan      bool
	Iperf         bool
	SNMP          bool
	AXFR          bool
	OpenRelayTest bool
	// ServerInfo exposes the server's interfaces/routes (CON-13); off in the public profile.
	ServerInfo bool
	// Speedtest serves bandwidth-test traffic (CON-09); off in the public profile.
	Speedtest       bool
	LAN             bool
	MetricsEnabled  bool
	LogHashIPs      bool
	AllowPrivate    bool
	TrustedProxies  []netip.Prefix
	DenyCIDRs       []netip.Prefix
	AllowCIDRs      []netip.Prefix
	GeoIPCityDB     string
	GeoIPASNDB      string
	RIPEstatSrcApp  string
	ProviderSecrets map[string]string
}

// Config is the full API configuration.
type Config struct {
	Profile   Profile
	Listen    string
	PublicURL string
	DataDir   string
	LogLevel  slog.Level
	Limits    Limits
	Features  Features
}

// providerSecretVars are read with *_FILE support and never logged.
var providerSecretVars = []string{
	"GLOBALPING_TOKEN", "IPINFO_TOKEN", "ABUSEIPDB_KEY", "PEERINGDB_KEY", "SSLLABS_EMAIL",
	"VIRUSTOTAL_KEY", "URLSCAN_KEY", "SAFEBROWSING_KEY", "MAXMIND_LICENSE_KEY",
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return LoadFrom(os.LookupEnv, os.ReadFile)
}

// ignoreComments treats a value that starts with "#" as unset. Docker Compose keeps an inline
// comment as the value when nothing precedes it (`ALLOW_CIDRS=   # if set, ...` in an env_file),
// and no setting legitimately starts with "#".
func ignoreComments(lookup LookupFunc) LookupFunc {
	return func(name string) (string, bool) {
		v, ok := lookup(name)
		if ok && strings.HasPrefix(strings.TrimSpace(v), "#") {
			return "", false
		}
		return v, ok
	}
}

// LookupFunc matches os.LookupEnv.
type LookupFunc func(string) (string, bool)

// ReadFileFunc matches os.ReadFile.
type ReadFileFunc func(string) ([]byte, error)

// LoadFrom reads configuration using the given lookup and file reader (injectable for tests).
func LoadFrom(lookup LookupFunc, readFile ReadFileFunc) (Config, error) {
	p := &parser{lookup: ignoreComments(lookup)}

	profile := Profile(strings.ToLower(p.str("NETTOOLS_PROFILE", string(ProfilePublic))))
	switch profile {
	case ProfilePublic, ProfileInternal, ProfileHomelab:
	default:
		return Config{}, fmt.Errorf("NETTOOLS_PROFILE: unknown profile %q (want public, internal or homelab)", profile)
	}
	relaxed := profile != ProfilePublic

	cfg := Config{
		Profile:   profile,
		Listen:    p.str("API_LISTEN", "0.0.0.0:8081"),
		PublicURL: p.str("PUBLIC_URL", "http://localhost:8080"),
		DataDir:   p.str("DATA_DIR", "/data"),
		LogLevel:  p.level("LOG_LEVEL", slog.LevelInfo),
		Limits: Limits{
			RateLimitRPM:       p.posInt("RATE_LIMIT_RPM", 60),
			RateLimitHeavyRPM:  p.posInt("RATE_LIMIT_HEAVY_RPM", 10),
			ConcurrencyGlobal:  p.posInt("CONCURRENCY_GLOBAL", 32),
			ConcurrencyPerIP:   p.posInt("CONCURRENCY_PER_IP", 3),
			PingMaxCount:       p.posInt("PING_MAX_COUNT", 20),
			PingMinInterval:    time.Duration(p.posInt("PING_MIN_INTERVAL_MS", 200)) * time.Millisecond,
			TracerouteMaxHops:  p.posInt("TRACEROUTE_MAX_HOPS", 30),
			MTRMaxCycles:       p.posInt("MTR_MAX_CYCLES", 20),
			PortscanMaxPorts:   p.posInt("PORTSCAN_MAX_PORTS", 1024),
			PortscanRatePPS:    p.posInt("PORTSCAN_RATE_PPS", 200),
			IperfMaxSeconds:    p.posInt("IPERF_MAX_SECONDS", 15),
			HTTPMaxBodyBytes:   int64(p.posInt("HTTP_MAX_BODY_BYTES", 1<<20)),
			ToolTimeoutDefault: p.duration("TOOL_TIMEOUT_DEFAULT", 15*time.Second),
			StreamTimeoutMax:   p.duration("STREAM_TIMEOUT_MAX", 120*time.Second),
			BulkMaxTargets:     p.posInt("BULK_MAX_TARGETS", 50),
		},
		Features: Features{
			Portscan:       p.boolean("PORTSCAN_ENABLED", relaxed),
			Iperf:          p.boolean("IPERF_ENABLED", true),
			SNMP:           p.boolean("SNMP_ENABLED", false),
			AXFR:           p.boolean("AXFR_ENABLED", true),
			OpenRelayTest:  p.boolean("OPEN_RELAY_TEST_ENABLED", false),
			ServerInfo:     p.boolean("SERVER_INFO_ENABLED", relaxed),
			Speedtest:      p.boolean("SPEEDTEST_ENABLED", relaxed),
			LAN:            p.boolean("NETTOOLS_LAN", false),
			MetricsEnabled: p.boolean("METRICS_ENABLED", false),
			LogHashIPs:     p.boolean("LOG_HASH_IPS", false),
			AllowPrivate:   p.boolean("ALLOW_PRIVATE_TARGETS", relaxed),
			TrustedProxies: p.prefixes("TRUSTED_PROXIES", "172.16.0.0/12"),
			DenyCIDRs:      p.prefixes("DENY_CIDRS", ""),
			AllowCIDRs:     p.prefixes("ALLOW_CIDRS", ""),
			GeoIPCityDB:    p.str("GEOIP_CITY_DB", ""),
			GeoIPASNDB:     p.str("GEOIP_ASN_DB", ""),
			RIPEstatSrcApp: p.str("RIPESTAT_SOURCEAPP", "nettools"),
		},
	}

	cfg.Features.ProviderSecrets = make(map[string]string, len(providerSecretVars))
	for _, name := range providerSecretVars {
		v, err := secret(p.lookup, readFile, name)
		if err != nil {
			p.errs = append(p.errs, err)
			continue
		}
		if v != "" {
			cfg.Features.ProviderSecrets[name] = v
		}
	}

	if err := errors.Join(p.errs...); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}
	return cfg, nil
}

// HasSecret reports whether a provider secret is configured, without exposing it.
func (c Config) HasSecret(name string) bool {
	return c.Features.ProviderSecrets[name] != ""
}

func secret(lookup LookupFunc, readFile ReadFileFunc, name string) (string, error) {
	if v, ok := lookup(name); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v), nil
	}
	path, ok := lookup(name + "_FILE")
	if !ok || strings.TrimSpace(path) == "" {
		return "", nil
	}
	b, err := readFile(strings.TrimSpace(path))
	if err != nil {
		// Do not include file contents; the path is not secret.
		return "", fmt.Errorf("%s_FILE: %w", name, err)
	}
	return strings.TrimSpace(string(b)), nil
}

type parser struct {
	lookup LookupFunc
	errs   []error
}

func (p *parser) raw(name string) (string, bool) {
	v, ok := p.lookup(name)
	v = strings.TrimSpace(v)
	return v, ok && v != ""
}

func (p *parser) str(name, def string) string {
	if v, ok := p.raw(name); ok {
		return v
	}
	return def
}

func (p *parser) posInt(name string, def int) int {
	v, ok := p.raw(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		p.errs = append(p.errs, fmt.Errorf("%s: want a positive integer, got %q", name, v))
		return def
	}
	return n
}

func (p *parser) boolean(name string, def bool) bool {
	v, ok := p.raw(name)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		p.errs = append(p.errs, fmt.Errorf("%s: want true or false, got %q", name, v))
		return def
	}
	return b
}

func (p *parser) duration(name string, def time.Duration) time.Duration {
	v, ok := p.raw(name)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		p.errs = append(p.errs, fmt.Errorf("%s: want a positive duration like 15s, got %q", name, v))
		return def
	}
	return d
}

func (p *parser) level(name string, def slog.Level) slog.Level {
	v, ok := p.raw(name)
	if !ok {
		return def
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(v)); err != nil {
		p.errs = append(p.errs, fmt.Errorf("%s: want debug, info, warn or error, got %q", name, v))
		return def
	}
	return l
}

func (p *parser) prefixes(name, def string) []netip.Prefix {
	v := p.str(name, def)
	var out []netip.Prefix
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			addr, err := netip.ParseAddr(part)
			if err != nil {
				p.errs = append(p.errs, fmt.Errorf("%s: invalid IP or CIDR %q", name, part))
				continue
			}
			out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		pfx, err := netip.ParsePrefix(part)
		if err != nil {
			p.errs = append(p.errs, fmt.Errorf("%s: invalid CIDR %q", name, part))
			continue
		}
		out = append(out, pfx.Masked())
	}
	return out
}
