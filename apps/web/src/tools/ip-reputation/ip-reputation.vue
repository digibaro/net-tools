<script setup lang="ts">
import type { AbuseReport } from './ip-reputation.service'
import { NButton, NProgress, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
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
import { isIpAddress, scoreLevel } from './ip-reputation.service'

const { t, te } = useI18n()
const ip = useQueryParam({ name: 'ip', defaultValue: '' })
const validation = useValidation({ source: ip, rules: [{ validator: isIpAddress, message: t('tools.reverse-dns.invalid') }] })
const { data, error, loading, run } = useApi<AbuseReport>('/v1/reputation/abuseipdb')
const submit = () => ip.value && validation.value.isValid && run({ ip: ip.value.trim() })
onMounted(() => {
  if (ip.value)
    void submit()
})
const levelType = { clean: 'success', suspicious: 'warning', malicious: 'error' } as const
const category = (n: number) => (te(`tools.ip-reputation.categories.${n}`) ? t(`tools.ip-reputation.categories.${n}`) : String(n))
const rows = computed(() => {
  const d = data.value
  if (!d)
    return []
  const k = (key: string) => t(`tools.ip-reputation.fields.${key}`)
  return [
    { label: k('reports'), value: t('tools.ip-reputation.reportCount', { n: d.totalReports, users: d.distinctUsers }), noCopy: true },
    { label: k('last'), value: d.lastReported || '—', mono: true },
    { label: k('isp'), value: [d.isp, d.domain].filter(Boolean).join(' · ') || '—' },
    { label: k('usage'), value: d.usageType || '—' },
    { label: k('country'), value: d.country || '—' },
    { label: k('flags'), value: [d.tor ? 'Tor' : '', d.whitelisted ? t('tools.ip-reputation.whitelisted') : ''].filter(Boolean).join(', ') || '—', noCopy: true },
  ]
})
const reportRows = computed(() => (data.value?.reports ?? []).map((r, i) => ({ id: i, when: r.reportedAt.slice(0, 16).replace('T', ' '), categories: r.categories.map(category).join(', '), comment: r.comment, country: r.country })))
const c = (key: string) => t(`tools.ip-reputation.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="ip" :label="t('tools.reverse-dns.ip')" placeholder="203.0.113.10" :validation="validation" monospace test-id="ip" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ip || !validation.isValid" data-testid="run">
        {{ t('bgp.lookUp') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="data.ip" data-testid="result">
      <template #header-extra>
        <NTag :type="levelType[scoreLevel(data.score)]" :bordered="false">
          {{ t(`tools.ip-reputation.level.${scoreLevel(data.score)}`) }}
        </NTag>
      </template>
      <div class="mb-3 max-w-md">
        <div class="mb-1 text-sm">
          {{ t('tools.ip-reputation.score', { score: data.score }) }}
        </div>
        <NProgress type="line" :percentage="data.score" :status="data.score >= 75 ? 'error' : data.score >= 25 ? 'warning' : 'success'" :show-indicator="false" :aria-label="t('tools.ip-reputation.score', { score: data.score })" />
      </div>
      <CKeyValueList :rows="rows" />
      <div class="mt-3 flex flex-wrap gap-2">
        <NButton tag="a" :href="data.url" target="_blank" rel="noopener noreferrer" size="small" secondary>
          {{ t('tools.ip-reputation.open') }}
        </NButton>
        <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" filename="abuseipdb.json" />
      </div>
    </CCard>
    <CCard v-if="reportRows.length" :title="t('tools.ip-reputation.recent')">
      <ResultTable :columns="[{ key: 'when', title: c('when'), mono: true, width: 150 }, { key: 'categories', title: c('categories'), width: 220 }, { key: 'comment', title: c('comment'), width: 320 }, { key: 'country', title: c('country'), width: 80 }]" :rows="reportRows" row-key="id" :max-height="400" :actions="false" />
    </CCard>
  </template>
</template>
