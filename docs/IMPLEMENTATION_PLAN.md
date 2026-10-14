# Implementation Plan

Work phase by phase. Each phase ends green (`make lint test e2e`), with the catalog updated, and a
tagged pre-release. Ask the user before starting the next phase if anything in the docs is ambiguous.

## Phase 0 — Scaffold (½–1 day)

- [x] Monorepo layout from `ARCHITECTURE.md`; pnpm workspace for `apps/web`; Go module `apps/api`.
- [x] Decide license (MIT vs GPL-3.0) with the user — **GPL-3.0-or-later**; add `LICENSE`, `NOTICE` (credit it-tools for UX inspiration).
- [x] `docker/web.Dockerfile`, `docker/api.Dockerfile`, `docker/nginx/default.conf.template`,
      `docker-compose.yml`, `docker-compose.dev.yml`, `.env.example`, `Makefile` (templates in this kit).
- [x] CI (GitHub Actions): lint, typecheck, unit tests, build images (amd64+arm64), Trivy, govulncheck.
- [x] `api`: config loader, slog JSON logger, chi router, envelope/errors, `/healthz`, `/readyz`,
      `/v1/version`, `/v1/capabilities` (empty), `/v1/echo`, `healthcheck` subcommand.
- **Exit**: `docker compose up` serves a blank SPA that calls `/api/v1/version`.

## Phase 1 — Shell + browser tools (2–4 days)

- [x] Layout copied in spirit from it-tools: `base.layout.vue` (sidebar + header), collapsible
      category menu, home grid with ToolCard, favorites store, fuse.js search + `Ctrl/Cmd+K` palette,
      About page, 404, i18n plugin, PWA.
- [x] **Theme light/dark/auto** per `THEMING.md` incl. anti-FOUC and Playwright theme tests.
- [x] `defineTool` with `runMode` + `RunModeBadge`; capabilities store + `ToolDisabled` component.
- [x] Shared components: `ResultTable`, `CopyButton`, `JsonDownload`, `ApiError`, `useQueryParam`,
      `useApi`, `useSse`, `StreamConsole`.
- [x] All **Phase 1** browser tools from the catalog (IP-*, DNS-13/14, WEB-12/13/14, TLS-05, CAL-*,
      CLI-04, IP-01).
- **Exit**: Lighthouse ≥ 90; every Phase 1 tool ✅.
  - Status (2026-10-13): all 32 Phase 1 tools ✅. Lighthouse accessibility / best practices / SEO
    100 on every page; performance 99 (desktop preset), 80 home / 72–75 tool pages (mobile
    preset: simulated slow 4G + 4× CPU). The remaining mobile gap is startup cost of Naive UI's
    runtime CSS-in-JS plus Vue/router/i18n init; closing it needs pre-rendering the shell or a
    lighter shell — pending a decision.
  - Status (2026-10-14, 150+ tools): mobile had dropped to 70 home / 69 tool page. Home cards now
    render in batches of 6 per idle period and the first navigation skips the forced scroll:
    79 home / 74 tool page (TBT 820 → 510 ms). Splitting the locale would add only ~2 points.
    The remaining gap is startup blocking (~500 ms at 4× CPU) and FCP 2.3 s; ≥ 90 needs
    pre-rendering or a lighter shell — pending a decision.

## Phase 2 — Core server tools (5–8 days)

- [x] `internal/target` + full SSRF test-suite **first**.
- [x] `internal/execx`, `internal/sse`, rate limiter, concurrency pool, per-tool timeouts.
- [x] DNS (DNS-01/02/03/04/06/09/10), Connectivity (CON-01/02/03/04/05/11/13), HTTP (WEB-01..04/06),
      TLS (TLS-01/02/06/07), Email (EM-01..04/07), Registry (BGP-01/06/07/08), Geo (REP-01 local mmdb),
      CLI-01/03, IP-06, DNS-12, CAL-05/10.
- [x] OpenAPI kept in sync; generated TS client.
- **Exit**: all Phase 2 rows ✅; `/security-review` checklist done per tool.

## Phase 3 — Public APIs & depth (4–6 days)

- [x] Provider framework: cache, key handling, `provider_key_missing` capability reasons.
- [x] Globalping (`remote/stream`) + "Worldwide" vantage switch in ping/traceroute/mtr/dns/http.
- [x] RIPEstat (BGP-02..05), PeeringDB (BGP-09), crt.sh (TLS-08), Observatory (WEB-04 option),
      HSTS preload (WEB-10), InternetDB/AbuseIPDB (REP-02/03), Tor (REP-04), cloud ranges (BGP-12).
- [x] REP-01 `source=ipinfo`: ipinfo.io provider (token required, `provider_key_missing` without it),
      used as fallback when no local GeoIP database is mounted. ip-api.com is not used (free tier is
      non-commercial only).
- [x] Remaining Phase 3 rows (DNSSEC, resolver bench, AXFR, wildcard, TLS scan, OCSP, STARTTLS, SSH
      audit, MTA-STS, DNSBL, PMTU, IPv6 readiness, latency matrix, WEB-05/07/08/09/11/15, CLI-02,
      speedtest profile CON-09).
- **Exit**: all Phase 3 rows ✅; provider tests use fixtures only.

## Phase 4 — Advanced & LAN (4–6 days)

- [x] `lan` compose override + LAN-* tools; capability reason `requires_lan_mode` when off.
- [x] Port scanner (CON-06), iperf3 (CON-08) + `iperf` profile, SNMP (CON-12), bgpq4 (BGP-10),
      AS-path visualizer, DMARC report viewer, open relay test, reputation (REP-05), CAL-12, TLS-11.
- **Exit**: all rows ✅ or ⏸ with a reason.

## Phase 5 — Hardening & release

- [ ] All-in-one image option; multi-arch release workflow pushing to GHCR with SBOM + provenance.
- [ ] Docs: README with screenshots (light + dark), configuration reference generated from config struct.
- [ ] Load test streaming endpoints (k6) to validate concurrency limits.
- [ ] v1.0.0 tag.

## Working agreements for Claude Code

- One tool (or one shared component) per commit; run tests before committing.
- When a tool needs a new dependency, prefer small, maintained libs; mention it in the commit body.
- Don't silently change limits, security rules or the stack — propose and ask.
- If a public API's behaviour differs from the catalog notes, update the catalog and tell the user.
