# NetTools — Claude Code starter kit

Specs and scaffolding for Claude Code to build **NetTools**: a self-hosted, Dockerized web app of
network tools and troubleshooting utilities, with the UX of
[it-tools](https://github.com/CorentinTh/it-tools/tree/v2024.10.22-7ca5933) and light / dark / auto theme.

## What's in here

| Path | Purpose |
|---|---|
| `CLAUDE.md` | Project memory Claude Code loads every session: stack, commands, hard rules |
| `docs/PRD.md` | Requirements and acceptance criteria |
| `docs/ARCHITECTURE.md` | Containers, repo layout, tool registry, data flow, libraries |
| `docs/TOOLS_CATALOG.md` | **122 tools** in 11 categories — run mode (browser / server / public API), implementation, public API option, phase, status |
| `docs/API_SPEC.md` | REST + SSE conventions, error codes, every endpoint |
| `docs/SECURITY.md` | SSRF policy, exec rules, limits, container hardening, review checklist |
| `docs/THEMING.md` | Three-state theme with Naive UI + UnoCSS, no-flash script, tests |
| `docs/TOOL_AUTHORING.md` | How to add one tool, frontend + backend |
| `docs/IMPLEMENTATION_PLAN.md` | Phases 0–5 with exit criteria |
| `docker-compose*.yml`, `docker/`, `.env.example`, `Makefile` | Runtime: `web` (nginx) + `api` (Go), profiles `iperf`, `speedtest`, LAN override |
| `.claude/settings.json` | Permission allow/deny list for Claude Code |
| `.claude/commands/` | Slash commands: `/implement-phase`, `/new-tool`, `/security-review`, `/catalog-sync` |

## How to use

```bash
claude
> /implement-phase 1
> /new-tool DNS-01        # or pick tools one by one
```

## Running (once built)

```bash
cp .env.example .env
docker compose up -d                                   # http://localhost:8080
docker compose --profile iperf --profile speedtest up -d
docker compose -f docker-compose.yml -f docker-compose.lan.yml up -d   # LAN tools (Linux host)
```

## License

GPL-3.0-or-later — see [LICENSE](LICENSE) and [NOTICE](NOTICE). UX inspired by
[it-tools](https://github.com/CorentinTh/it-tools).
