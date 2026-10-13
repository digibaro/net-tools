---
description: Review server-side code changes against docs/SECURITY.md
argument-hint: [path or tool id, default = uncommitted changes]
---

Review ${ARGUMENTS:-the current uncommitted changes} for security, using `docs/SECURITY.md`.

For every handler touched, verify and report pass/fail with file:line evidence:
- Target obtained only via `target.Resolve`, pinned IP used for dialing (HTTP redirects re-resolved).
- All params validated, numeric ones clamped to config limits, enums checked.
- Timeout, rate-limit class and concurrency slot applied.
- No shell; argv built by builder; no user value can start with `-`.
- Response/body/output size bounded; secrets never logged.
- SSRF table test present (private, loopback, link-local, metadata, mapped IPv6, odd IPv4 notations,
  DNS rebinding).
- Capability registered with correct disabled reasons.

Fix what fails (ask first if a fix changes behaviour or limits), then re-run the tests.
