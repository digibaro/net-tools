<script setup lang="ts">
import type { RoaCoverage } from './roa-coverage.service'
import { NAlert, NButton, NProgress } from 'naive-ui'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import CopyButton from '@/components/CopyButton.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { parseAsn, prefixPath } from '@/utils/bgp'
import { fixed } from '@/utils/format'
import { coverage, suggestions } from './roa-coverage.service'

const { t } = useI18n()
const asn = useQueryParam({ name: 'asn', defaultValue: '' })
const input = ref(asn.value)
const validation = useValidation({ source: input, rules: [{ validator: (v: string) => parseAsn(v) !== undefined, message: t('bgp.invalidAsn') }] })
const { data, error, loading, run } = useApi<RoaCoverage>('/v1/bgp/roa-coverage')
function submit() {
  const n = parseAsn(input.value)
  if (n === undefined)
    return
  if (asn.value === `AS${n}`)
    void run({}, `/AS${n}`)
  asn.value = `AS${n}`
}
watch(asn, (v) => {
  input.value = v
  const n = parseAsn(v)
  if (n !== undefined)
    void run({}, `/AS${n}`)
}, { immediate: true })
const pct = computed(() => (data.value ? coverage(data.value) : 0))
const order = { 'invalid': 0, 'not-found': 1, 'unchecked': 2, 'valid': 3 }
const prefixRows = computed(() => [...(data.value?.prefixes ?? [])]
  .sort((a, b) => order[a.status] - order[b.status])
  .map(p => ({ id: p.prefix, prefix: p.prefix, path: prefixPath(p.prefix), status: t(`tools.roa-coverage.status.${p.status}`) + (p.reason ? ` (${t(`tools.roa-coverage.reason.${p.reason}`)})` : ''), roa: p.roa || '—' })))
const roaRows = computed(() => (data.value?.roas ?? []).map(r => ({ id: `${r.prefix}-${r.maxLength}`, prefix: r.prefix, maxLength: r.maxLength, ta: r.ta, note: !r.used ? t('tools.roa-coverage.unused') : r.loose ? t('tools.roa-coverage.loose') : '✓' })))
const advice = computed(() => (data.value ? suggestions(data.value) : []))
const k = (key: string) => t(`tools.roa-coverage.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="input" :label="t('bgp.asn')" placeholder="AS3333" :validation="validation" monospace test-id="asn" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!validation.isValid" data-testid="run">
        {{ t('tools.roa-coverage.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="`AS${data.asn} ${data.name}`" data-testid="result">
      <div class="mb-1 text-sm" data-testid="coverage">
        {{ t('tools.roa-coverage.summary', { pct: fixed(pct, 1), valid: data.valid, invalid: data.invalid, notFound: data.notFound, total: data.prefixes.length }) }}
      </div>
      <NProgress type="line" :percentage="pct" :show-indicator="false" :aria-label="t('tools.roa-coverage.coverage')" class="mb-3 max-w-md" />
      <NAlert v-if="data.invalid" type="error" class="mb-3">
        {{ t('tools.roa-coverage.invalidWarning', { n: data.invalid }) }}
      </NAlert>
      <NAlert v-if="data.unchecked" type="info" class="mb-3">
        {{ t('tools.roa-coverage.uncheckedNote', { n: data.unchecked }) }}
      </NAlert>
      <ResultTable :columns="[{ key: 'prefix', title: k('prefix'), mono: true, width: 200, routeKey: 'path' }, { key: 'status', title: k('status'), width: 220 }, { key: 'roa', title: k('roa'), mono: true, width: 220 }]" :rows="prefixRows" row-key="id" filename="roa-coverage.json" />
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.roa-coverage.source', { validator: data.validator, time: data.generated }) }}
      </p>
    </CCard>
    <CCard v-if="advice.length" :title="t('tools.roa-coverage.suggest')">
      <p class="mt-0 text-sm">
        {{ t('tools.roa-coverage.suggestHelp') }}
      </p>
      <div class="flex items-start gap-2">
        <pre class="m-0 min-w-0 flex-1 overflow-auto text-xs font-mono" tabindex="0">{{ advice.join('\n') }}</pre>
        <CopyButton :text="advice.join('\n')" :label="t('common.copyText')" />
      </div>
    </CCard>
    <CCard v-if="roaRows.length" :title="t('tools.roa-coverage.roas', { n: roaRows.length })">
      <ResultTable :columns="[{ key: 'prefix', title: k('prefix'), mono: true, width: 200 }, { key: 'maxLength', title: 'maxLength', width: 100 }, { key: 'ta', title: k('ta'), width: 200 }, { key: 'note', title: k('note'), width: 260 }]" :rows="roaRows" row-key="id" filename="roas.json" />
    </CCard>
  </template>
</template>
