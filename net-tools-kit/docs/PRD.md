# Product Requirements — NetTools

## 1. Summary

NetTools is a self-hosted, Docker-delivered web app that gathers network utilities (calculators,
lookups, live diagnostics, internet-registry queries) behind one fast, searchable UI. It copies the
proven UX of it-tools — sidebar of categories, global search, favorites, one page per tool — and adds
a backend so tools that need real sockets (ping, traceroute, DNS, TLS, port checks) can run from the
server's vantage point, plus optional public APIs for an outside-in / worldwide view.

## 2. Users

| Persona | Needs |
|---|---|
| Homelab / self-hoster | Quick LAN + WAN diagnostics from a box on their network |
| Sysadmin / network engineer | Subnetting, DNS/TLS/email checks, BGP/ASN lookups, traceroute from server and from the world |
| Developer / SRE | HTTP headers, TLS expiry, CORS, DNS propagation during deploys |
| Helpdesk | Simple "is it down / what's my IP / ping this" with shareable results |

## 3. Goals

1. One UI for ~90 network tools (see `TOOLS_CATALOG.md`), shipped incrementally by phase.
2. Three execution modes, visible per tool:
   - **Browser** — pure client-side computation, works offline (PWA).
   - **Server** — runs inside the `api` container (the server's network vantage point).
   - **Public API** — proxied through the server to third-party services (global probes, BGP data, CT logs…).
3. Runs as `docker compose up` with sensible defaults; one or multiple containers.
4. Theme: **light, dark, auto** (follows OS, live-updates on OS change), persisted per browser.
5. Safe by default when exposed on the internet (SSRF protection, rate limits, feature flags).

## 4. Non-goals (v1)

- User accounts / multi-tenant (rely on a reverse proxy with auth if needed).
- Persistent monitoring / alerting / history (maybe v2: "saved checks").
- Packet capture in the browser.
- Replacing nmap/Wireshark; we expose bounded, safe subsets.

## 5. Functional requirements

### 5.1 Shell (mirrors it-tools)
- FR-1 Collapsible left sidebar grouped by category; categories collapsible; favorites pinned on top.
- FR-2 Home page: grid of tool cards (favorites first, then "newest", then all), with run-mode badge.
- FR-3 Global fuzzy search (fuse.js on name, description, keywords); `Ctrl/Cmd+K` opens command palette.
- FR-4 Each tool at `/<tool-path>`; inputs mirrored to URL query params so results are shareable.
- FR-5 Favorite toggle per tool (localStorage).
- FR-6 Theme switcher in header: Light / Dark / Auto (see `THEMING.md`).
- FR-7 i18n with English as source; locale switcher; missing keys fall back to English.
- FR-8 PWA: browser-only tools work offline.
- FR-9 About page: version, build commit, enabled capabilities, third-party attributions.
- FR-10 Tools whose capability is disabled on the server are shown greyed with a reason
  ("Disabled by administrator", "Requires API key", "Requires LAN mode").

### 5.2 Tool UX contract
- FR-11 Validate input inline before submit; show normalized target (e.g. resolved IP) in results.
- FR-12 Streaming tools (ping, traceroute, mtr, port scan, iperf) show live output and a Stop button.
- FR-13 Every result: copy as text, copy as JSON, download JSON; tables sortable.
- FR-14 Server tools show "Run from: This server ▾ / Worldwide (Globalping)" when both exist.
- FR-15 Errors are human-readable with the error code from the API (e.g. `TARGET_PRIVATE_DENIED`).

### 5.3 Backend
- FR-16 REST JSON under `/api/v1`, SSE for streaming (`API_SPEC.md`).
- FR-17 `GET /api/v1/capabilities` drives which tools are enabled in the UI.
- FR-18 Feature flags and limits via env vars (`.env.example`).
- FR-19 Third-party API keys configured server-side only.
- FR-20 Optional response cache (in-memory, TTL per provider) for public API calls.

## 6. Non-functional requirements

| Area | Requirement |
|---|---|
| Performance | Initial JS < 300 KB gzip for shell; tools lazy-loaded; API p95 < 300 ms for non-network work |
| Security | `SECURITY.md` fully applied; container passes Trivy with no critical CVEs |
| Accessibility | WCAG 2.1 AA contrast in both themes; keyboard navigable; focus visible |
| Compatibility | Last 2 versions of Chrome, Firefox, Safari, Edge; mobile ≥360px |
| Images | Multi-arch (`linux/amd64`, `linux/arm64`); web image < 50 MB; api image < 150 MB |
| Observability | Structured JSON logs; `/api/healthz`, `/api/readyz`; optional Prometheus `/metrics` |
| Licensing | Project license MIT or GPL-3.0 (decide in Phase 0). it-tools is GPL-3.0: take inspiration from layout, do **not** copy code verbatim unless you adopt GPL-3.0 |

## 7. Acceptance criteria (release v1.0)

- `docker compose up -d` → app on `:8080`, healthchecks green within 30 s.
- All Phase 1–3 tools in the catalog marked ✅, each with tests.
- Theme: switching light/dark/auto works without reload; auto follows OS change live; no flash of
  wrong theme on load; Playwright test covers all three.
- SSRF test-suite passes (private, loopback, link-local, metadata IP, DNS-rebinding cases).
- Lighthouse ≥ 90 for performance, accessibility, best practices on the home page.
