---
description: Reconcile docs/TOOLS_CATALOG.md status with what is actually implemented
---

1. List tool folders in `apps/web/src/tools/` and tools registered in `apps/web/src/tools/index.ts`.
2. List backend tools registered in `apps/api/cmd/nettools-api/main.go` and their capability ids.
3. Compare with `docs/TOOLS_CATALOG.md`: fix Status columns, and report mismatches
   (implemented but not in catalog, marked ✅ but missing tests, capability id mismatch).
4. Recompute the Totals table. Do not change anything else.
