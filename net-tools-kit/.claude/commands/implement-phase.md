---
description: Work through one phase of docs/IMPLEMENTATION_PLAN.md (e.g. /implement-phase 1)
argument-hint: <phase-number>
---

Implement **Phase $ARGUMENTS** of `docs/IMPLEMENTATION_PLAN.md`.

1. Read `CLAUDE.md`, the phase section, and every doc it references.
2. Build a task list: shared infrastructure items first, then each catalog row of this phase
   (filter `docs/TOOLS_CATALOG.md` by Phase = $ARGUMENTS and Status ≠ ✅).
3. Do infrastructure items, then each tool following `/new-tool` steps — one commit per item.
4. After every ~5 tools run `make lint test`; fix before continuing.
5. At the end check the phase **Exit** criteria, run `make e2e`, and give a short report:
   done, deferred (⏸ with reason), and anything that needs a decision from me.
