<script setup lang="ts">
import type { PrefixStatus } from './prefix-status.service'
import { NAlert, NButton, NProgress, NTag } from 'naive-ui'
import { computed, watch } from 'vue'
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
import { asnPath, prefixPath } from '@/utils/bgp'
import { isPrefixOrIp, visibilityPct } from './prefix-status.service'

const { t } = useI18n()
const prefix = useQueryParam({ name: 'prefix', defaultValue: '' })
const validation = useValidation({ source: prefix, rules: [{ validator: isPrefixOrIp, message: t('bgp.invalidPrefix') }] })
const { data, error, loading, run } = useApi<PrefixStatus>('/v1/prefix/status')
const submit = () => prefix.value && validation.value.isValid && run({ prefix: prefix.value.trim() })
watch(prefix, () => void submit(), { immediate: true })
const rows = computed(() => {
  const d = data.value
  if (!d)
    return []
  const k = (key: string) => t(`bgp.fields.${key}`)
  return [
    { label: k('visibility'), value: t('bgp.visibility', { seen: d.seenBy, total: d.totalPeers, pct: visibilityPct(d) }), noCopy: true },
    { label: k('firstSeen'), value: d.firstSeen ? `${d.firstSeen.time} (AS${d.firstSeen.origin})` : '—', mono: true },
    { label: k('lastSeen'), value: d.lastSeen ? `${d.lastSeen.time} (AS${d.lastSeen.origin})` : '—', mono: true },
  ]
})
const originRows = computed(() => (data.value?.origins ?? []).map(o => ({ asn: `AS${o.asn}`, link: asnPath(o.asn), objects: o.routeObjects.join(', ') || '—' })))
const related = (list: PrefixStatus['lessSpecifics']) => list.map(r => ({ prefix: r.prefix, link: prefixPath(r.prefix), origin: r.origin ? `AS${r.origin}` : '—' }))
const c = (key: string) => t(`bgp.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="prefix" :label="t('bgp.prefix')" placeholder="1.1.1.0/24" :validation="validation" monospace test-id="prefix" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!prefix || !validation.isValid" data-testid="run">
        {{ t('bgp.lookUp') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-if="data.moas" type="warning" class="mb-4" data-testid="moas">
      {{ t('tools.prefix-status.moas', { n: data.origins.length }) }}
    </NAlert>
    <CCard :title="data.prefix" data-testid="result">
      <template #header-extra>
        <NTag :type="data.announced ? 'success' : 'default'" :bordered="false">
          {{ data.announced ? t('bgp.announced') : t('bgp.notAnnounced') }}
        </NTag>
      </template>
      <NProgress type="line" :percentage="visibilityPct(data)" :show-indicator="false" :aria-label="t('bgp.fields.visibility')" class="mb-3 max-w-md" />
      <CKeyValueList :rows="rows" />
      <h3 class="mb-2 mt-4 text-sm font-semibold">
        {{ t('tools.prefix-status.origins') }}
      </h3>
      <ResultTable :columns="[{ key: 'asn', title: 'ASN', mono: true, routeKey: 'link', width: 120 }, { key: 'objects', title: c('routeObjects'), width: 260 }]" :rows="originRows" row-key="asn" :actions="false" />
      <div class="mt-3">
        <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" filename="prefix-status.json" />
      </div>
    </CCard>
    <CCard v-if="data.lessSpecifics.length || data.moreSpecifics.length" :title="t('tools.prefix-status.related')">
      <h3 v-if="data.lessSpecifics.length" class="mb-2 mt-0 text-sm font-semibold">
        {{ t('tools.prefix-status.less') }}
      </h3>
      <ResultTable v-if="data.lessSpecifics.length" :columns="[{ key: 'prefix', title: c('prefix'), mono: true, routeKey: 'link' }, { key: 'origin', title: c('origin'), width: 120 }]" :rows="related(data.lessSpecifics)" row-key="prefix" :actions="false" />
      <h3 v-if="data.moreSpecifics.length" class="mb-2 mt-4 text-sm font-semibold">
        {{ t('tools.prefix-status.more') }}
      </h3>
      <ResultTable v-if="data.moreSpecifics.length" :columns="[{ key: 'prefix', title: c('prefix'), mono: true, routeKey: 'link' }, { key: 'origin', title: c('origin'), width: 120 }]" :rows="related(data.moreSpecifics)" row-key="prefix" :max-height="320" :actions="false" />
    </CCard>
  </template>
</template>
