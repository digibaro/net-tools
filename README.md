# NET-TOOLS

Self-hosted network tools and troubleshooting utilities in one web app — 125 tools for IP
addressing, DNS, connectivity, HTTP, TLS, email, BGP, reputation and your LAN. The UX follows
[it-tools](https://github.com/CorentinTh/it-tools): a searchable grid of tools, each on its own
page, light / dark / auto theme, shareable links. Ships as two Docker containers.

| Category | Tools | Examples |
|---|---:|---|
| IP & Addressing | 19 | IPv4/IPv6 subnet calculators, CIDR aggregation, MAC/OUI lookup, IPv6 ULA |
| DNS | 14 | Lookup, reverse, propagation, DNSSEC, trace, zone transfer test, DNS benchmark |
| Connectivity & Diagnostics | 14 | Ping, TCP ping, traceroute, MTR, port check/scan, path MTU, iperf3, SNMP |
| HTTP & Web | 15 | HTTP headers, redirects, security headers, CORS, robots/sitemap |
| TLS & Certificates | 11 | TLS inspector, cipher scan, expiry, CSR/certificate decoder, JWK/PEM/SSH converter |
| Email | 9 | MX/SMTP, SPF, DMARC, DKIM, MTA-STS, header analyzer, DMARC reports, open relay test |
| Routing, BGP & Registry | 12 | IP→ASN, WHOIS, RDAP, PeeringDB, IRR / AS-SET expansion, AS-path visualizer |
| Reputation & Geolocation | 5 | IP geolocation, AbuseIPDB, passive exposure, Tor exit check, URL reputation |
| LAN | 6 | Wake-on-LAN, ARP scan, mDNS and SSDP discovery, DHCP probe, gateway / NAT info |
| Calculators & Reference | 12 | Bandwidth, data units, port reference, router config snippets |
| Client Diagnostics | 5 | Request headers, browser latency, WebRTC leak test, browser ↔ server speed test |

Every tool shows where it runs:

- **Browser** — computed in your browser; nothing is sent anywhere.
- **Server** — the NET-TOOLS server does the work from *its* network (ping, traceroute, scans…).
- **Public API** — a third-party service is queried through the server (keys stay on the server).

<img width="1312" height="824" alt="Image" src="https://github.com/user-attachments/assets/420bafa8-e29b-4914-9452-ca6995514483" />

## Quick start

Requirements: Docker with Compose v2. LAN mode needs a Linux host.

```bash
git clone https://github.com/digibaro/net-tools.git
cd net-tools
cp .env.example .env          # optional: settings and API keys (see Configuration)
docker compose up -d --build  # build from source
```

Open <http://localhost:8080>. To use the published images instead of building, pull them for a
release (`NETTOOLS_VERSION` selects the image tag, e.g. `0.4.0`; default `latest`):

```bash
docker compose pull && docker compose up -d
```

Stop with `docker compose down`. To update: `git pull`, then the same `up` command again.

## Configuration

All settings are environment variables, read from `.env` (copy it from `.env.example`). After
changing `.env`, recreate the api container — a plain restart keeps the old values:

```bash
docker compose up -d --force-recreate api
```

> **Never put real keys in `.env.example`** — it is part of the repository. `.env` is ignored by
> git. A key can also come from a file: `ABUSEIPDB_KEY_FILE=/run/secrets/abuseipdb` (Docker secrets).

### Profiles

`NETTOOLS_PROFILE` sets safe defaults for where you run NET-TOOLS; individual variables override
them.

| Setting | `public` (default) | `internal` / `homelab` |
|---|---|---|
| Targets in private ranges (`ALLOW_PRIVATE_TARGETS`) | denied | allowed |
| Port scanner (`PORTSCAN_ENABLED`) | off | on |
| Browser ↔ server speed test (`SPEEDTEST_ENABLED`) | off | on |
| Server network info (`SERVER_INFO_ENABLED`) | off | on |

Use `public` for anything reachable from the Internet. Cloud metadata endpoints and
special-purpose ranges (link-local, multicast, reserved) are blocked in every profile.

### Feature switches

| Variable | Default | Tool |
|---|---|---|
| `PORTSCAN_ENABLED` | by profile | Port scanner (CON-06) |
| `SPEEDTEST_ENABLED` | by profile | Browser ↔ server speed test |
| `SERVER_INFO_ENABLED` | by profile | Server network info |
| `IPERF_ENABLED` | `true` | iperf3 bandwidth test |
| `AXFR_ENABLED` | `true` | DNS zone transfer test |
| `SNMP_ENABLED` | `false` | SNMP get / walk |
| `OPEN_RELAY_TEST_ENABLED` | `false` | Open relay test (never sends a message) |
| `NETTOOLS_LAN` | `false` | LAN tools — set by the LAN override, see below |

A disabled tool stays visible with the reason (disabled by admin, needs LAN mode, API key
missing…), so users know why it does not run.

### API keys (optional)

Tools that need a key show "API key missing" until it is set.

| Variable | Used by | Get one at |
|---|---|---|
| `ABUSEIPDB_KEY` | IP reputation | <https://www.abuseipdb.com/account/api> |
| `VIRUSTOTAL_KEY` | URL / domain reputation | <https://www.virustotal.com/gui/my-apikey> |
| `URLSCAN_KEY` | URL / domain reputation | <https://urlscan.io/user/profile/> |
| `SAFEBROWSING_KEY` | URL / domain reputation | Google Cloud console, Safe Browsing API |
| `IPINFO_TOKEN` | IP geolocation (when no GeoIP database is mounted) | <https://ipinfo.io/account/token> |
| `PEERINGDB_KEY` | PeeringDB (works without; raises limits) | <https://www.peeringdb.com/> |
| `GLOBALPING_TOKEN` | Worldwide ping/traceroute/DNS (works without; raises limits) | <https://dash.globalping.io/> |

URL reputation needs at least one of its three keys; the others are reported as skipped.

### Limits

Rate limits and per-tool bounds keep the server from being abused. The most useful ones:

| Variable | Default | Meaning |
|---|---|---|
| `RATE_LIMIT_RPM` | 60 | Requests per minute per client IP |
| `RATE_LIMIT_HEAVY_RPM` | 10 | Streaming, scanning and provider tools per minute per client IP |
| `CONCURRENCY_PER_IP` / `CONCURRENCY_GLOBAL` | 3 / 32 | Simultaneous heavy requests |
| `PORTSCAN_MAX_PORTS` / `PORTSCAN_RATE_PPS` | 1024 / 200 | Port scanner bounds |
| `IPERF_MAX_SECONDS` | 15 | Longest iperf3 test |

The complete list with explanations is in [docs/SECURITY.md](docs/SECURITY.md).

### GeoIP databases

IP geolocation works offline with a MaxMind GeoLite2 or DB-IP Lite database placed in `data/`
(mounted read-only). See [data/README.md](data/README.md).

## Optional services

```bash
docker compose --profile iperf up -d       # iperf3 server on port 5201 (IPERF_PORT)
docker compose --profile speedtest up -d   # LibreSpeed on port 8082 (SPEEDTEST_PORT)
```

The built-in speed test tool measures browser ↔ NET-TOOLS server without LibreSpeed; the
profile adds LibreSpeed's own UI for comparison.

## LAN mode

Wake-on-LAN, ARP scan, mDNS and SSDP discovery, DHCP probe and gateway / NAT info need to see your
local network, so both containers use the host's network:

```bash
docker compose -f docker-compose.yml -f docker-compose.lan.yml up -d --build   # or: make up-lan
```

- Linux hosts only (Docker Desktop on macOS/Windows does not expose the LAN this way).
- The web UI listens on port 8080 of the host; the api stays on host loopback and is not
  reachable from the LAN.
- Only `NET_RAW` is added (ARP, DHCP probe and ICMP); no `NET_ADMIN`.
- Use it on trusted networks only, typically with `NETTOOLS_PROFILE=homelab`.

Without the override these tools show "needs LAN mode".

## Behind a reverse proxy / HTTPS

Put NET-TOOLS behind your proxy (Traefik, Caddy, nginx) and forward to port 8080.

- Set `PUBLIC_URL` to the public address (used in the User-Agent of outgoing requests).
- Set `TRUSTED_PROXIES` to the proxy's address range so client IPs (for rate limits) are taken
  from `X-Forwarded-For`.
- Set `ENABLE_HSTS=true` once HTTPS works.

Health checks: `GET /api/healthz` (process up) and `GET /api/readyz` (dependencies ready).

## Using it

- Search with <kbd>Ctrl</kbd>+<kbd>K</kbd>, star tools to pin them on the home page.
- Every input is in the URL — copy the address to share a result.
- Settings (gear icon): theme (light / dark / auto), accent colour, language.
- Server tools run from the server's location; many DNS and connectivity tools can also run
  **Worldwide** through Globalping probes.

## Security

NET-TOOLS runs network operations on behalf of its users, so the api validates every target
(no private ranges in the public profile, DNS pinning against rebinding), runs CLI tools only from
an allow-list without a shell, limits every tool, and keeps provider keys on the server. Details
and the review checklist: [docs/SECURITY.md](docs/SECURITY.md).

## Development

```bash
make dev        # hot reload on http://localhost:8080 (web and api in Docker)
make test       # web unit + api unit tests
make lint       # eslint, vue-tsc, golangci-lint
make up-e2e && make e2e   # Playwright against the running stack (relaxed rate limits)
make new-tool name=my-tool   # scaffold a tool
```

`make up` builds from source and stamps the version from git (`0.4.0` on a release tag,
`0.4.0-3-gabc1234` after it); a plain `docker compose up --build` shows `<version>-dev`. To run a
release instead, use `docker compose pull && docker compose up -d` (no `--build`).

Frontend: Vue 3, Vite, TypeScript, Naive UI, UnoCSS (`apps/web`). Backend: Go, chi (`apps/api`).

| Document | Contents |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Containers, repository layout, tool registry, data flow |
| [docs/TOOLS_CATALOG.md](docs/TOOLS_CATALOG.md) | Every tool with run mode, implementation and status |
| [docs/API_SPEC.md](docs/API_SPEC.md) | REST + SSE conventions, error codes, endpoints (`apps/api/openapi.yaml`) |
| [docs/SECURITY.md](docs/SECURITY.md) | Target policy, exec rules, limits, container hardening |
| [docs/THEMING.md](docs/THEMING.md) | Theme modes, accent colours, contrast rules |
| [docs/TOOL_AUTHORING.md](docs/TOOL_AUTHORING.md) | Adding a tool, frontend and backend |
| [docs/PRD.md](docs/PRD.md) · [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md) | Requirements and phases |

## License

GPL-3.0-or-later — © 2026 digibaro. See [LICENSE](LICENSE) and [NOTICE](NOTICE). UX inspired by
[it-tools](https://github.com/CorentinTh/it-tools).
