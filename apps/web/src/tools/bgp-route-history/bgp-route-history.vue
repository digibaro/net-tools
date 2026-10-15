<script setup lang="ts">
import type { RouteHistory } from './bgp-route-history.service'
import { NAlert, NButton, NProgress } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { asnPath, isPrefixOrIp } from '@/utils/bgp'
import { fixed } from '@/utils/format'
import { day, originVerdict, visibility } from './bgp-route-history.service'

const { t } = useI18n()
const resource = useQueryParam({ name: 'resource', defaultValue: '' })
const validation = useValidation({ source: resource, rules: [{ validator: (s: string) => !s || isPrefixOrIp(s), message: t('tools.bgp-route-history.invalid') }] })
const { data, error, loading, run } = useApi<RouteHistory>('/v1/bgp/route-history')
const submit = () => resource.value && validation.value.isValid && run({ resource: resource.value.trim() })
onMounted(() => {
  if (resource.value)
    void submit()
})
const vis = computed(() => (data.value ? visibility(data.value) : undefined))
const v = computed(() => (data.value ? originVerdict(data.value) : undefined))
const alertType = { none: 'warning', single: 'success', moas: 'warning' } as const
const originRows = computed(() => (data.value?.origins ?? []).flatMap(o => o.prefixes.map(p => ({
  id: `${o.asn}|${p.prefix}`,
  asn: `AS${o.asn}`,
  asnPath: asnPath(o.asn),
  name: o.name || '—',
  prefix: p.prefix,
  period: `${day(p.firstSeen)} – ${p.current ? t('tools.bgp-route-history.now') : day(p.lastSeen)}`,
  peers: fixed(p.maxPeers, 0),
}))))
const collectorRows = computed(() => (data.value?.collectors ?? []).filter(c => c.fullTables > 0).map(c => ({ id: c.name, name: c.name, where: [c.city, c.country].filter(Boolean).join(', '), seeing: `${c.fullTables - c.notSeeing} / ${c.fullTables}` })))
const k = (key: string) => t(`tools.bgp-route-history.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="resource" :label="t('tools.bgp-route-history.resource')" placeholder="193.0.0.0/21" :validation="validation" monospace test-id="resource" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!resource || !validation.isValid" data-testid="run">
        {{ t('tools.bgp-route-history.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data && v">
    <CCard :title="data.resource" data-testid="result">
      <NAlert :type="alertType[v]" class="mb-3" data-testid="verdict">
        {{ t(`tools.bgp-route-history.verdict.${v}`, { n: data.origins.filter(o => o.current).length }) }}
      </NAlert>
      <div v-if="vis !== undefined" class="mb-1 text-sm">
        {{ t('tools.bgp-route-history.visibility', { pct: fixed(vis, 1), seeing: data.seeing, total: data.fullTablePeers, family: data.family }) }}
      </div>
      <NProgress v-if="vis !== undefined" type="line" :percentage="vis" :show-indicator="false" :aria-label="t('tools.bgp-route-history.visibilityLabel')" class="mb-3 max-w-md" />
      <h3 class="mb-2 mt-2 text-sm font-semibold">
        {{ t('tools.bgp-route-history.origins', { from: day(data.queryStart) }) }}
      </h3>
      <ResultTable :columns="[{ key: 'asn', title: 'ASN', width: 100, routeKey: 'asnPath' }, { key: 'name', title: k('name'), width: 240 }, { key: 'prefix', title: k('prefix'), mono: true, width: 180 }, { key: 'period', title: k('period'), width: 220 }, { key: 'peers', title: k('peers'), width: 110 }]" :rows="originRows" row-key="id" filename="route-history.json" />
    </CCard>
    <CCard v-if="collectorRows.length" :title="t('tools.bgp-route-history.collectors')">
      <ResultTable :columns="[{ key: 'name', title: k('collector'), width: 100 }, { key: 'where', title: k('where'), width: 200 }, { key: 'seeing', title: k('seeing'), width: 160 }]" :rows="collectorRows" row-key="id" filename="ris-visibility.json" />
    </CCard>
  </template>
</template>
