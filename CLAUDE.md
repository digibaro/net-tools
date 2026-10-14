# CLAUDE.md — NetTools

Self-hosted web app that bundles network tools and troubleshooting utilities, in the spirit of
[it-tools](https://github.com/CorentinTh/it-tools/tree/v2024.10.22-7ca5933) (same UX, layout and
per-tool folder convention), but focused on networking. Ships as Docker containers.

Read these before doing non-trivial work:

| Doc | Read when |
|---|---|
| `docs/PRD.md` | Scope, goals, non-goals, acceptance criteria |
| `docs/ARCHITECTURE.md` | Repo layout, containers, data flow, tool registry |
| `docs/TOOLS_CATALOG.md` | The full tool list, run mode, phase, status — the backlog |
| `docs/API_SPEC.md` | Backend endpoints, envelopes, SSE streaming, error codes |
| `docs/SECURITY.md` | SSRF rules, rate limits, exec rules, container hardening — **mandatory for any server tool** |
| `docs/THEMING.md` | Light / dark / auto theme implementation |
| `docs/TOOL_AUTHORING.md` | Step-by-step for adding one tool (frontend + backend) |
| `docs/IMPLEMENTATION_PLAN.md` | Phases, order of work, definition of done |

## Stack (do not change without asking)

- **Frontend** `apps/web`: Vue 3 + Vite + TypeScript, Naive UI, UnoCSS, Pinia, vue-router,
  vue-i18n, @vueuse/core, fuse.js (search), pnpm. Tests: Vitest (unit) + Playwright (e2e).
  Use current stable versions, not the versions pinned in it-tools.
- **Backend** `apps/api`: Go (latest stable), `chi` router, `log/slog`, `miekg/dns`,
  `prometheus-community/pro-bing`, `openrdap`, stdlib `crypto/tls`. SSE for streaming tools.
- **Runtime**: `web` (nginx, static SPA + reverse proxy `/api`) and `api` (Go + a few allow-listed
  CLI binaries). Optional compose profiles: `iperf`, `speedtest`, `lan`.

## Commands

```bash
make dev            # docker compose -f docker-compose.yml -f docker-compose.dev.yml up (hot reload)
make up / make down # production-like stack on http://localhost:8080
make test           # web unit + api unit
make e2e            # Playwright against the running stack (start it with `make up-e2e`: relaxed rate limits)
make lint           # eslint + vue-tsc + golangci-lint
make new-tool name=dns-lookup   # scaffolds a tool (see docs/TOOL_AUTHORING.md)

# inside apps/web
pnpm dev | pnpm build | pnpm test | pnpm test:e2e | pnpm lint | pnpm typecheck
# inside apps/api
go test ./... | go vet ./... | golangci-lint run | go run ./cmd/nettools-api
```

## Hard rules

1. **Never run a shell.** Server tools call binaries via `internal/execx` with an argv array from an
   allow-list. No `sh -c`, no string concatenation into commands.
2. **Every network target goes through `internal/target.Resolve()`** (validation, DNS resolution,
   private-range policy, IP pinning). No handler may dial a user-supplied host directly.
3. **Every server tool has limits**: timeout, max count/ports/hops, and a concurrency slot. Defaults
   live in `internal/config` and are documented in `docs/SECURITY.md`.
4. **API keys for public services never reach the browser.** Keyed providers are called only from
   `apps/api/internal/providers/*`. The browser may call keyless, CORS-enabled APIs directly only if
   the catalog marks it so.
5. **Prefer browser-only implementations** for pure computation (subnetting, converters, decoders).
   Don't add a backend endpoint for something JS can do.
6. **Each tool = its own folder** (`apps/web/src/tools/<kebab-name>/`), registered in
   `apps/web/src/tools/index.ts`, with a `*.service.ts` holding logic and a `*.service.test.ts`.
7. **Theme**: never hard-code colors in components; use Naive UI theme vars or UnoCSS tokens so
   light, dark and auto all work. Check both themes before marking a tool done.
8. **Update `docs/TOOLS_CATALOG.md`** status column when a tool is implemented.
9. Keep `apps/api/openapi.yaml` in sync with handlers; the frontend client types are generated from it.
10. All user-facing strings go through vue-i18n (`apps/web/locales/en.yml` is the source).

## Code style

- TS: strict mode, no `any`, composition API with `<script setup lang="ts">`, antfu eslint config.
- Go: small packages, errors wrapped with `%w`, context everywhere, table-driven tests,
  no global state except config.
- Commits: Conventional Commits (`feat(dns-lookup): …`, `fix(api): …`).

## Definition of done (per tool)

- Works in light and dark theme, mobile width (≥360px) and desktop.
- Input validation with inline errors; results copyable; shareable via URL query params.
- Unit tests for service logic; for server tools also a handler test with a fake target.
- Server tools: limits enforced, SSRF tests pass, appears in `/api/v1/capabilities`.
- Catalog row updated to `✅`.
