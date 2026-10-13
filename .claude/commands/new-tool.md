---
description: Implement one tool from docs/TOOLS_CATALOG.md end-to-end (e.g. /new-tool DNS-01)
argument-hint: <catalog-id>
---

Implement catalog tool **$ARGUMENTS**.

1. Read its row in `docs/TOOLS_CATALOG.md`, then `docs/TOOL_AUTHORING.md`. If run mode is S/P/H/L,
   also read `docs/SECURITY.md` and the matching section of `docs/API_SPEC.md`.
2. Set the row status to 🚧.
3. Scaffold with `make new-tool name=<kebab-name>` (add `server=true` when a backend is needed).
4. Implement service logic first with unit tests, then the backend handler (if any) with SSRF and
   limit tests, then the Vue component.
5. Update `openapi.yaml` and run `make gen-api` if you added endpoints.
6. Verify: `make lint test`; check light, dark and auto theme and a 360px-wide viewport.
7. For server tools, go through the checklist in `docs/SECURITY.md` §10 and report each item.
8. Set the row status to ✅ and commit `feat(<kebab-name>): <summary>`.

Stop and ask if the catalog row is ambiguous or a public API behaves differently than documented.
