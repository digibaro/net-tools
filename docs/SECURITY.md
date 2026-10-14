# Security — NetTools

A network-tools server is, by design, a machine that makes outbound connections to user-chosen
targets. Without guardrails it becomes an SSRF pivot into your LAN/cloud metadata, a scanner, or a
DoS amplifier. Everything here is **mandatory** for server-side tools.

## 1. Deployment profiles

| Profile | `ALLOW_PRIVATE_TARGETS` | Port scan | Rate limit | Typical use |
|---|---|---|---|---|
| `public` (default) | `false` | off | strict | Exposed on the internet |
| `internal` | `true` | on (bounded) | relaxed | Behind VPN / SSO, team use |
| `homelab` | `true` + `lan` profile | on | relaxed | Single user on home network |

Set with `NETTOOLS_PROFILE=public|internal|homelab`; individual env vars override profile defaults.
The server network info tool (CON-13, `SERVER_INFO_ENABLED`) reveals the server's interfaces and
routes and is therefore off in `public` and on in the other profiles.
The UI shows the active profile on the About page.

## 2. Target validation — `internal/target`

`Resolve(ctx, input string, opts) (Target, error)` is the **only** way to obtain an address to dial.

1. Parse: accept hostname (IDNA → ASCII, max 253 chars, label rules), IPv4, IPv6 (with optional zone
   **rejected**), `host:port` only where the tool allows. Reject URLs' userinfo, non-http(s) schemes.
2. Resolve hostnames with the server resolver (timeout 3 s). Respect `family`.
3. **Policy check on every resolved IP** (`net/netip`):
   - Always deny: `0.0.0.0/8`, `::/128`, `169.254.169.254/32`, `fd00:ec2::254/128`,
     `100.100.100.200/32` (Alibaba), `168.63.129.16/32` (Azure), multicast, broadcast,
     `DENY_CIDRS` env list.
   - Deny unless `ALLOW_PRIVATE_TARGETS=true`: RFC 1918, `100.64.0.0/10`, `127.0.0.0/8`, `::1`,
     `fc00::/7`, `fe80::/10`, `169.254.0.0/16`, IPv4-mapped forms of all of the above,
     6to4/NAT64-embedded private v4, the container's own addresses.
   - Optional `ALLOW_CIDRS` allow-list mode (only these targets).
4. **Pin** the chosen IP: tools dial `Target.IP`, never re-resolve the hostname (DNS-rebinding
   defense). For HTTP tools use a custom `DialContext` that ignores the host and dials the pinned IP;
   on redirects, run `Resolve` again for the new host (max 10 redirects).
5. TLS tools pass the original hostname as SNI but dial the pinned IP.

Required tests (`internal/target/target_test.go`): every deny range for v4/v6, mapped addresses,
decimal/octal/hex IPv4 forms (`2130706433`, `0177.0.0.1`, `0x7f.1`), trailing dot, uppercase,
IDN homographs normalized, a fake resolver returning public then private IP (rebinding).

## 3. Command execution — `internal/execx`

- Allow-list map of tool → absolute binary path, checked at startup (`binary_missing` capability).
- Args built as `[]string`; user values only in positions defined by a per-tool builder; never
  allow values starting with `-`; numeric params parsed to ints and clamped.
- `exec.CommandContext`, process group, kill on ctx cancel, `Cmd.WaitDelay`.
- Minimal env (`PATH`, `LANG=C`), no stdin, stdout line limit (e.g. 1 MiB total), stderr captured.
- `nmap`: only `-sT -Pn --top-ports N` or `-p <validated list>`, `--max-rate`, no scripts (`--script`
  never), no OS detection, no `-oN` file output.

## 4. Limits (defaults, env-overridable)

| Var | Default (public) | Meaning |
|---|---|---|
| `RATE_LIMIT_RPM` | 60 | Requests/min per client IP (token bucket) |
| `RATE_LIMIT_HEAVY_RPM` | 10 | For streaming/scan/provider tools |
| `CONCURRENCY_GLOBAL` | 32 | Simultaneous heavy jobs |
| `CONCURRENCY_PER_IP` | 3 | Simultaneous heavy jobs per client |
| `PING_MAX_COUNT` | 20 | |
| `PING_MIN_INTERVAL_MS` | 200 | |
| `TRACEROUTE_MAX_HOPS` | 30 | |
| `MTR_MAX_CYCLES` | 20 | |
| `PORTSCAN_MAX_PORTS` | 1024 | Also `PORTSCAN_ENABLED=false` by default in public |
| `PORTSCAN_RATE_PPS` | 200 | |
| `IPERF_MAX_SECONDS` | 15 | |
| `HTTP_MAX_BODY_BYTES` | 1 MiB | Response body read cap |
| `TOOL_TIMEOUT_DEFAULT` | 15 s | Per request; streaming max 120 s |
| `BULK_MAX_TARGETS` | 50 | For bulk tools (expiry, propagation) |

Fixed limits for tools that follow data from third parties (not configurable):

| Tool | Limit |
|---|---|
| SPF (EM-02) | Walk stops after 10 DNS-querying terms (RFC 7208 permerror), include depth 12 |
| DMARC (EM-03) | 10 report URIs per tag checked for external authorization |
| MX / SMTP (EM-01) | 10 MX hosts, 64 KiB and 100 lines per reply read before STARTTLS, 15 s per probe |
| WHOIS (BGP-06) | 3 referrals, ports 43 and 4321 only, 256 KiB per answer, 10 s per server |
| RDAP / providers | `netx.PolicyClient`: policy on every dial incl. bootstrap targets and redirects, 5 MiB body |

Client IP for rate limiting comes from `X-Forwarded-For` **only if** the immediate peer is in
`TRUSTED_PROXIES` (default: the `web` container network).

## 5. Container hardening

`api`:
- `cap_drop: [ALL]`, `cap_add: [NET_RAW]` (ICMP/traceroute). `NET_ADMIN` only in `lan` profile if
  DHCP probe is enabled.
- Runs as root **inside the container** with only `NET_RAW`, `read_only: true`, `tmpfs: /tmp`,
  `security_opt: [no-new-privileges:true]`. Alternative (stricter, more work): non-root user +
  `setcap cap_net_raw+ep` on `mtr-packet`/`traceroute`/api binary and drop `no-new-privileges`
  (file caps need it off). Pure-Go ICMP can avoid raw sockets entirely using
  `sysctls: net.ipv4.ping_group_range: "0 2147483647"` — keep it set.
- No Docker socket, no host mounts except read-only `/data` (GeoIP, OUI).
- Resource limits: `mem_limit: 256m`, `pids_limit: 256`, `cpus: 1.0`.

`web`:
- `nginxinc/nginx-unprivileged`, port 8080, read-only FS with tmpfs for cache/run.
- Headers: `Content-Security-Policy` (`default-src 'self'; connect-src 'self' https://api.ipify.org https://api6.ipify.org https://cloudflare-dns.com https://dns.google ws: wss:; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'sha256-<theme-script-hash>'; frame-ancestors 'none'`),
  `ws: wss:` in connect-src is for the browser WebSocket tester (WEB-11): connections are opened only
  on an explicit user action and carry no NetTools credentials.
  `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `Permissions-Policy`,
  HSTS only when behind TLS (`ENABLE_HSTS`).
- `proxy_buffering off` + `X-Accel-Buffering: no` for SSE.

## 6. Secrets & providers

- Provider keys via env or Docker secrets (`*_FILE` variants supported): `GLOBALPING_TOKEN`,
  `IPINFO_TOKEN`, `ABUSEIPDB_KEY`, `PEERINGDB_KEY`, `SSLLABS_EMAIL`, `VIRUSTOTAL_KEY`,
  `URLSCAN_KEY`, `SAFEBROWSING_KEY`, `MAXMIND_LICENSE_KEY` (only for the optional updater).
- Never log keys or full upstream URLs containing keys. Redact in errors.
- Outbound provider calls: timeout 10 s (30 s for crt.sh, which is often slow), response size cap 5 MiB, user-agent
  `NetTools/<version> (+<PUBLIC_URL>)`.

## 7. Auth

No built-in accounts in v1. For anything beyond `homelab`, put NetTools behind an authenticating
reverse proxy (Authelia, oauth2-proxy, Cloudflare Access, Tailscale). Optional built-in
`BASIC_AUTH_USERS` (htpasswd bcrypt) in nginx for quick protection.

## 8. Logging & audit

- JSON logs: request id, client ip (or hashed if `LOG_HASH_IPS=true`), tool, target input,
  resolved ip, outcome, duration. Never request bodies of SNMP/HTTP tools (may contain creds).
- Denied-target attempts logged at `warn` for abuse detection.

## 9. Supply chain

- Pin base images by digest in release builds; Renovate/Dependabot enabled.
- CI: `go vet`, `golangci-lint`, `govulncheck`, `pnpm audit --prod`, Trivy image scan, SBOM (syft).

## 10. `/security-review` checklist (run before merging any server tool)

- [ ] Target goes through `target.Resolve` and the pinned IP is used.
- [ ] Every numeric param clamped; every enum validated.
- [ ] Timeout + concurrency slot + rate class set.
- [ ] No shell; argv built by builder; no user value can become a flag.
- [ ] Response size bounded.
- [ ] SSRF table test added for the new handler.
- [ ] Capability registered with correct disabled reasons.
