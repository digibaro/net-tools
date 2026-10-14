<script setup lang="ts">
import type { AsnInfo } from './asn-info.service'
import { NButton, NTag } from 'naive-ui'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { prefixPath } from '@/utils/bgp'
import { neighbourRows, parseAsn } from './asn-info.service'

const { t } = useI18n()
// The URL holds the last looked-up ASN; typing edits a local copy so no request runs per keystroke.
const asn = useQueryParam({ name: 'asn', defaultValue: '' })
const input = ref(asn.value)
const validation = useValidation({ source: input, rules: [{ validator: (v: string) => parseAsn(v) !== undefined, message: t('bgp.invalidAsn') }] })
const parsed = computed(() => parseAsn(input.value))
const { data, error, loading, run } = useApi<AsnInfo>('/v1/asn')
function submit() {
  if (parsed.value === undefined)
    return
  if (asn.value === `AS${parsed.value}`)
    void run({}, `/AS${parsed.value}`) // same ASN again: the watcher would not fire
  asn.value = `AS${parsed.value}`
}
// Runs on load, on submit and when a neighbour link changes the query string.
watch(asn, (v) => {
  input.value = v
  const n = parseAsn(v)
  if (n !== undefined)
    void run({}, `/AS${n}`)
}, { immediate: true })
const rows = computed(() => (data.value
  ? [
      { label: t('bgp.fields.holder'), value: data.value.holder },
      { label: t('bgp.fields.registry'), value: data.value.block || '—' },
      { label: t('bgp.fields.prefixes'), value: t('bgp.prefixCount', { v4: data.value.prefixesV4.length, v6: data.value.prefixesV6.length, total: data.value.prefixesTotal }), noCopy: true },
      { label: t('bgp.fields.neighbours'), value: t('bgp.neighbourCount', { up: data.value.upstreams, down: data.value.downstreams, total: data.value.neighboursTotal }), noCopy: true },
    ]
  : []))
const relation = (type: string) => t(`bgp.relation.${type}`)
const prefixRows = computed(() => [...(data.value?.prefixesV4 ?? []), ...(data.value?.prefixesV6 ?? [])].map(p => ({ prefix: p, link: prefixPath(p) })))
const c = (key: string) => t(`bgp.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="input" :label="t('bgp.asn')" placeholder="AS13335" :validation="validation" monospace test-id="asn" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="parsed === undefined" data-testid="run">
        {{ t('bgp.lookUp') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="`AS${data.asn}`" data-testid="result">
      <template #header-extra>
        <NTag :type="data.announced ? 'success' : 'default'" :bordered="false">
          {{ data.announced ? t('bgp.announced') : t('bgp.notAnnounced') }}
        </NTag>
      </template>
      <CKeyValueList :rows="rows" />
      <div class="mt-3">
        <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" :filename="`as${data.asn}.json`" />
      </div>
    </CCard>
    <CCard v-if="prefixRows.length" :title="t('bgp.prefixesTitle', { n: data.prefixesTotal })">
      <ResultTable :columns="[{ key: 'prefix', title: c('prefix'), mono: true, routeKey: 'link' }]" :rows="prefixRows" row-key="prefix" :max-height="360" :filename="`as${data.asn}-prefixes.json`" />
    </CCard>
    <CCard v-if="data.neighbours.length" :title="t('bgp.neighboursTitle')">
      <ResultTable :columns="[{ key: 'asn', title: 'ASN', mono: true, routeKey: 'link', width: 120 }, { key: 'relation', title: c('relation'), width: 160 }, { key: 'power', title: c('power'), width: 100, sortable: true }]" :rows="neighbourRows(data, relation)" row-key="asn" :max-height="360" :filename="`as${data.asn}-neighbours.json`" />
      <p class="mb-0 text-xs nt-muted">
        {{ t('bgp.powerNote') }}
      </p>
    </CCard>
  </template>
</template>
