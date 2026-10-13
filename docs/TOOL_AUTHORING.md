# Tool Authoring Guide

How to add one tool end-to-end. `make new-tool name=<kebab-name>` (→ `apps/web/scripts/create-tool.mjs`)
scaffolds steps 1–3; with `server=true` it also scaffolds the Go package (step 5).

## 1. Pick the row in `TOOLS_CATALOG.md`

Note the ID, run mode, capability id, phase. Set status to 🚧.

## 2. Frontend folder

```
apps/web/src/tools/dns-lookup/
├── index.ts
├── dns-lookup.vue
├── dns-lookup.service.ts
├── dns-lookup.service.test.ts
└── dns-lookup.e2e.spec.ts
```

```ts
// index.ts
import { World } from '@vicons/tabler'
import { defineTool } from '../tool'
import { translate } from '@/plugins/i18n.plugin'

export const tool = defineTool({
  name: translate('tools.dns-lookup.title'),
  path: '/dns-lookup',
  description: translate('tools.dns-lookup.description'),
  keywords: ['dns', 'dig', 'nslookup', 'record', 'mx', 'txt', 'resolver'],
  component: () => import('./dns-lookup.vue'),
  icon: World,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  capability: 'dns.lookup',
})
```

Register in `src/tools/index.ts` under the right category.

## 3. Component pattern

```vue
<script setup lang="ts">
import { useQueryParam } from '@/composables/queryParams'
import { useApi } from '@/composables/useApi'
import { isValidHostname, RECORD_TYPES } from './dns-lookup.service'

const name = useQueryParam({ name: 'name', defaultValue: '' })
const type = useQueryParam({ name: 'type', defaultValue: 'A' })
const resolver = useQueryParam({ name: 'resolver', defaultValue: 'system' })

const { data, error, loading, run } = useApi('/api/v1/dns/lookup')
const nameValidation = useValidation({ source: name, rules: [{ validator: isValidHostname, message: t('…') }] })
</script>

<template>
  <c-card>
    <c-input-text v-model:value="name" :label="t('…')" :validation="nameValidation" @keyup.enter="run({ name, type, resolver })" />
    <c-select v-model:value="type" :options="RECORD_TYPES" />
    <c-button :loading="loading" :disabled="!nameValidation.isValid" @click="run({ name, type, resolver })">Lookup</c-button>
  </c-card>
  <api-error v-if="error" :error="error" />
  <result-table v-if="data" :rows="data.answers" copyable downloadable />
</template>
```

Rules:
- Logic (parsing, validation, formatting) lives in `*.service.ts` and is unit-tested.
- Use `c-*` UI wrappers and theme vars only (see `THEMING.md`).
- Inputs ↔ URL query params via `useQueryParam` so results are shareable.
- Streaming tools use `useSse()` + `StreamConsole` / typed tables, and expose a Stop button.
- If `capabilities.tools[cap].enabled === false`, the tool page shows `<tool-disabled :reason>`.
- Hybrid tools show a vantage selector: **This server** / **Worldwide** (only if `remote.globalping` enabled).

## 4. i18n

Add `tools.<name>.title`, `.description` and UI strings to `apps/web/locales/en.yml`.

## 5. Backend package (server / hybrid / public-api tools)

```
apps/api/internal/tools/dnslookup/
├── register.go     # implements tools.Tool: ID(), Mount(), Enabled()
├── handler.go      # HTTP parsing → service call → envelope
├── service.go      # pure-ish logic with injected resolver/dialer
└── service_test.go # table-driven; fake DNS server via miekg/dns test server
```

```go
func (t *Tool) Mount(r chi.Router, d tools.Deps) {
    r.With(d.RateLimit(ratelimit.Light), d.Timeout(5*time.Second)).
        Get("/dns/lookup", t.handleLookup)
}

func (t *Tool) handleLookup(w http.ResponseWriter, r *http.Request) {
    q, err := parseLookupQuery(r.URL.Query())          // validates & clamps
    if err != nil { httpapi.Error(w, r, httpapi.InvalidInput(err)); return }
    res, err := t.svc.Lookup(r.Context(), q)           // resolver IP passed through target.Resolve
    httpapi.Respond(w, r, res, err)
}
```

Streaming handler skeleton:

```go
func (t *Tool) handleStream(w http.ResponseWriter, r *http.Request) {
    q, err := parse(r.URL.Query()); if err != nil { … }
    tgt, err := t.deps.Target.Resolve(r.Context(), q.Target, target.Opts{Family: q.Family})
    if err != nil { httpapi.Error(w, r, err); return }
    release, err := t.deps.Pool.Acquire(r.Context(), clientIP(r)); if err != nil { … }
    defer release()
    s := sse.New(w); defer s.Close()
    s.Event("meta", meta(tgt, q))
    err = t.svc.Run(r.Context(), tgt, q, func(ev any) { s.Event(eventName(ev), ev) })
    if err != nil { s.Event("error", httpapi.ToAPIError(err)); return }
    s.Event("done", summary)
}
```

Then:
1. Add the tool to the registry list in `cmd/nettools-api/main.go`.
2. Document the endpoint in `openapi.yaml`; run `pnpm gen:api` in `apps/web`.
3. Add limits to `internal/config` + `.env.example` + `SECURITY.md` table if new.
4. Add SSRF cases to the handler test.

## 6. Public-API providers

```
apps/api/internal/providers/ripestat/
├── client.go        # base URL, timeout, user-agent, cache, sourceapp param
├── types.go
└── client_test.go   # httptest server with recorded fixtures (testdata/*.json)
```

Never hit the real API in unit tests. Optional `-tags=live` integration tests.

## 7. Done

Run `/security-review` (server tools), `make lint test`, check both themes + mobile width,
update catalog row to ✅, commit `feat(<tool>): …`.
