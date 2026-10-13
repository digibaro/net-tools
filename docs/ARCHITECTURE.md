# Architecture — NetTools

## 1. Container topology

```
                 ┌──────────────────────────── docker compose ────────────────────────────┐
 Browser ──:8080─▶  web  (nginx, unprivileged)                                            │
                 │   • serves SPA (apps/web/dist)                                         │
                 │   • /api/*  ──proxy (buffering off for SSE)──▶  api :8081               │
                 │                                                  (Go, cap NET_RAW)     │
                 │                                                   │ native: DNS, TCP,  │
                 │                                                   │ ICMP, TLS, HTTP,   │
                 │                                                   │ WHOIS/RDAP         │
                 │                                                   │ exec (allow-list): │
                 │                                                   │ mtr, traceroute,   │
                 │                                                   │ iperf3, nmap*      │
                 │                                                   ▼                    │
                 │                                   Public APIs (Globalping, RIPEstat,   │
                 │                                   crt.sh, RDAP, PeeringDB, …)          │
                 │  optional profiles:                                                    │
                 │   • iperf      → iperf3 server :5201 (test target for clients)          │
                 │   • speedtest  → LibreSpeed :8082 (browser ↔ server bandwidth)         │
                 │   • lan        → api with network_mode: host (ARP, WoL, mDNS, SSDP)      │
                 └────────────────────────────────────────────────────────────────────────┘
```

- **Single-container option**: `docker/allinone.Dockerfile` (Phase 5) runs nginx + api under `tini`
  with `s6`-less supervision (api as child of a tiny Go launcher) for users who want one image.
- The **vantage point matters**: server tools measure from the Docker host's network. LAN-layer tools
  (ARP scan, Wake-on-LAN, mDNS/SSDP discovery) only work in `lan` profile (host networking).

## 2. Repository layout

```
net-tools/
├── CLAUDE.md
├── LICENSE  NOTICE                # GPL-3.0-or-later; it-tools credited for UX
├── Makefile
├── package.json  pnpm-workspace.yaml  pnpm-lock.yaml   # pnpm workspace (apps/web)
├── .github/workflows/ci.yml      # lint, typecheck, tests, multi-arch images, Trivy, e2e
├── data/                         # mounted read-only at /data in api (GeoLite2 mmdb, …)
├── docker-compose.yml            # web + api (+ profiles)
├── docker-compose.dev.yml        # vite dev server + air (Go hot reload)
├── docker-compose.lan.yml        # override: api on host network
├── .env.example
├── docker/
│   ├── web.Dockerfile
│   ├── api.Dockerfile
│   └── nginx/default.conf.template  security-headers.conf
├── docs/                         # this kit
├── .claude/
│   ├── settings.json
│   └── commands/                 # /new-tool, /implement-phase, /security-review, /catalog-sync
├── apps/
│   ├── web/                      # ── mirrors it-tools layout ──
│   │   ├── index.html            # contains anti-FOUC theme script
│   │   ├── vite.config.ts  uno.config.ts  tsconfig*.json  playwright.config.ts
│   │   ├── locales/en.yml …
│   │   ├── scripts/create-tool.mjs
│   │   └── src/
│   │       ├── main.ts  App.vue  router.ts  config.ts
│   │       ├── layouts/base.layout.vue  tool.layout.vue
│   │       ├── pages/Home.page.vue  About.page.vue  NotFound.page.vue  Settings.page.vue
│   │       ├── components/       # NavbarButtons, ThemeSwitcher, CommandPalette, ToolCard,
│   │       │                     # CollapsibleToolMenu, RunModeBadge, StreamConsole, ResultTable
│   │       ├── ui/               # c-button, c-card, c-input-text, c-select, c-copy … (thin wrappers)
│   │       ├── composables/      # useQueryParam, useCopy, useApi, useSse, useValidation
│   │       ├── stores/           # style.store.ts (theme), tool.store.ts (favorites), capabilities.store.ts
│   │       ├── api/              # generated types from apps/api/openapi.yaml + fetch client
│   │       ├── themes.ts         # Naive UI light/dark overrides
│   │       └── tools/
│   │           ├── tool.ts       # defineTool()
│   │           ├── index.ts      # categories → tools
│   │           └── <tool-name>/
│   │               ├── index.ts
│   │               ├── <tool-name>.vue
│   │               ├── <tool-name>.service.ts
│   │               ├── <tool-name>.service.test.ts
│   │               └── <tool-name>.e2e.spec.ts
│   └── api/
│       ├── go.mod
│       ├── openapi.yaml
│       ├── cmd/nettools-api/main.go         # also `nettools-api healthcheck`
│       └── internal/
│           ├── config/          # env parsing, limits, feature flags
│           ├── httpapi/         # router, envelope, error codes, capabilities handler
│           ├── middleware/      # request id, logging, ratelimit, concurrency, cors, recover
│           ├── target/          # Resolve(): parse, resolve, policy, pinning  (SECURITY critical)
│           ├── execx/           # allow-listed command runner, line streaming, kill on ctx
│           ├── sse/             # SSE writer, heartbeats
│           ├── cache/           # TTL cache for providers
│           ├── providers/       # globalping, ripestat, crtsh, rdap, peeringdb, ipinfo, shodan, …
│           └── tools/<tool>/    # handler.go, service.go, service_test.go, register.go
└── e2e/ (optional, if e2e kept outside apps/web)
```

## 3. Tool registry (frontend)

it-tools defines each tool with `defineTool({...})`. We extend it with runtime metadata:

```ts
// apps/web/src/tools/tool.ts
export type RunMode = 'browser' | 'server' | 'public-api' | 'hybrid'

export interface ToolDefinition {
  name: string                 // i18n key resolved in index.ts
  path: string                 // '/dns-lookup'
  description: string
  keywords: string[]
  component: () => Promise<Component>
  icon: Component              // @vicons/tabler
  createdAt?: Date             // drives "new" badge
  runMode: RunMode
  capability?: string          // e.g. 'dns.lookup' — must be true in /capabilities
  providers?: string[]         // e.g. ['globalping'] for "Worldwide" option
  requiresLan?: boolean
}
export const defineTool = (t: ToolDefinition) => ({ ...t, isNew: isNew(t.createdAt) })
```

```ts
// apps/web/src/tools/index.ts
export const toolsByCategory: ToolCategory[] = [
  { name: 'IP & Addressing', components: [ipv4SubnetCalculator, ipv6SubnetCalculator, ...] },
  { name: 'DNS', components: [dnsLookup, reverseDns, ...] },
  { name: 'Connectivity', components: [ping, traceroute, ...] },
  { name: 'HTTP & Web', components: [...] },
  { name: 'TLS & Certificates', components: [...] },
  { name: 'Email', components: [...] },
  { name: 'Routing, BGP & Registry', components: [...] },
  { name: 'Reputation & Geo', components: [...] },
  { name: 'LAN', components: [...] },
  { name: 'Calculators & Reference', components: [...] },
  { name: 'Client Diagnostics', components: [...] },
]
```

Categories and order are the same as in `TOOLS_CATALOG.md`.

## 4. Backend tool registration

```go
// internal/tools/registry.go
type Tool interface {
    ID() string                         // "dns.lookup"
    Mount(r chi.Router, deps Deps)      // registers its routes under /api/v1
    Enabled(cfg config.Config) (bool, string) // reason when disabled
}
```

`main.go` builds `[]Tool`, mounts enabled ones, and `/api/v1/capabilities` reports all of them with
`enabled` + `reason`. Shared deps: config, logger, target resolver, exec runner, rate limiter,
concurrency pool, provider clients, cache.

## 5. Data flow examples

**Browser tool (IPv4 subnet calculator)**: input → `service.ts` pure function → render. No network.

**Server tool, request/response (DNS lookup)**:
`GET /api/v1/dns/lookup?name=example.com&type=MX&resolver=1.1.1.1` → ratelimit → `target` policy on
resolver IP → `miekg/dns` query with timeout → envelope JSON.

**Server tool, streaming (traceroute)**:
`EventSource('/api/v1/traceroute/stream?target=…&proto=icmp')` → resolve + policy → acquire
concurrency slot → `execx.Run(ctx, "mtr", argv…)` or native → each hop emitted as `event: hop` →
`event: done` with summary → slot released. Client closing the EventSource cancels ctx → process killed.

**Public API (Globalping)**: `POST /api/v1/remote/measurements` (our API) → provider client adds
token if configured → polls Globalping → streams results via SSE in the same hop/reply schema as local
tools so the UI component is shared.

## 6. Key libraries

| Concern | Frontend | Backend |
|---|---|---|
| IP math | `ip-address` or `ipaddr.js` (+ BigInt for v6) | `net/netip` |
| DNS | — (DoH via fetch where allowed) | `github.com/miekg/dns` |
| ICMP ping | — | `github.com/prometheus-community/pro-bing` |
| Traceroute/MTR | — | `mtr --json`/`--raw` via execx (native Go fallback later) |
| TLS | `@peculiar/x509` / `pkijs` for decoding pasted certs | `crypto/tls`, `crypto/x509`, `golang.org/x/crypto/ocsp` |
| WHOIS / RDAP | — | `github.com/likexian/whois` + `github.com/openrdap/rdap` |
| GeoIP | — | `github.com/oschwald/maxminddb-golang` (GeoLite2 mmdb mounted at `/data`) |
| OUI | `oui-data` (bundled) | — |
| Charts (latency) | `echarts` via `vue-echarts` (lazy) | — |
| WireGuard keys | `@noble/curves` (x25519) | — |
