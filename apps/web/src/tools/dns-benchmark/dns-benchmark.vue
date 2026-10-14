<script setup lang="ts">
import type { BenchResolver, BenchSummary } from './dns-benchmark.service'
import { NButton, NFormItem, NInputNumber, NProgress, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { liveOrder, successPct } from './dns-benchmark.service'

const { t } = useI18n()
const set = useQueryParam({ name: 'set', defaultValue: 'quick' })
const custom = useQueryParam({ name: 'custom', defaultValue: '' })
const names = useQueryParam({ name: 'names', defaultValue: '' })
const rounds = useQueryParam({ name: 'rounds', defaultValue: 2 })
const results = ref<BenchResolver[]>([])
const expected = ref(0)
const summary = shallowRef<BenchSummary>()
const { status, error, start, stop } = useSse('/v1/dns/benchmark/stream', {
  meta: d => (expected.value = (d as { resolvers: number }).resolvers),
  resolver: d => results.value.push(d as BenchResolver),
  done: d => (summary.value = (d as { summary: BenchSummary }).summary),
})
const running = computed(() => status.value === 'running')
function run() {
  results.value = []
  summary.value = undefined
  start({ set: set.value, custom: custom.value || undefined, names: names.value || undefined, rounds: rounds.value })
}
const ordered = computed(() => summary.value?.ranking ?? liveOrder(results.value))
const fastest = computed(() => ordered.value.find(r => r.answered > 0)?.medianMs ?? 0)
const rows = computed(() => ordered.value.map((r, i) => ({
  rank: r.answered ? i + 1 : '—',
  name: r.name,
  median: r.answered ? r.medianMs : '—',
  avg: r.answered ? r.avgMs : '—',
  p95: r.answered ? r.p95Ms : '—',
  success: `${successPct(r)} %`,
  errors: r.errors.join('; '),
})))
const c = (key: string) => t(`tools.dns-benchmark.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="flex flex-wrap items-end gap-x-6">
        <NFormItem :label="t('tools.dns-benchmark.set')" :show-feedback="false" class="mb-3">
          <NRadioGroup v-model:value="set" name="set">
            <NRadioButton value="quick" :label="t('tools.dns-benchmark.quick')" />
            <NRadioButton value="full" :label="t('tools.dns-benchmark.full')" />
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('tools.dns-benchmark.rounds')" :show-feedback="false" class="mb-3 w-28">
          <NInputNumber v-model:value="rounds" :min="1" :max="3" :show-button="false" :input-props="{ 'aria-label': t('tools.dns-benchmark.rounds') }" />
        </NFormItem>
      </div>
      <CInputText v-model:value="custom" :label="t('tools.dns-benchmark.custom')" placeholder="192.0.2.53, 9.9.9.11" monospace test-id="custom" />
      <CInputText v-model:value="names" :label="t('tools.dns-benchmark.names')" :placeholder="t('tools.dns-benchmark.namesPlaceholder')" monospace test-id="names" />
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="running" data-testid="run">
          {{ t('tools.dns-benchmark.run') }}
        </NButton>
        <NButton v-if="running" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="results.length || running" :title="t('common.result')" data-testid="result">
    <NProgress v-if="running" type="line" :percentage="expected ? Math.round(results.length / expected * 100) : 0" :show-indicator="false" :aria-label="t('tools.dns-benchmark.progress')" class="mb-3" />
    <div class="mb-4 flex flex-col gap-1">
      <div v-for="r in ordered.filter(x => x.answered)" :key="r.name" class="grid grid-cols-[minmax(8rem,14rem)_1fr_5rem] items-center gap-2 text-xs">
        <span class="truncate" :title="r.name">{{ r.name }}</span>
        <NProgress type="line" :percentage="Math.min(100, Math.round(fastest / Math.max(r.medianMs, 0.001) * 100))" :show-indicator="false" :aria-label="r.name" />
        <span class="text-right font-mono">{{ r.medianMs }} ms</span>
      </div>
    </div>
    <ResultTable :columns="[{ key: 'rank', title: '#', width: 50 }, { key: 'name', title: c('resolver'), width: 220 }, { key: 'median', title: c('median'), width: 90, sortable: true }, { key: 'avg', title: c('avg'), width: 80 }, { key: 'p95', title: 'p95', width: 80 }, { key: 'success', title: c('success'), width: 90 }, { key: 'errors', title: c('errors'), width: 220 }]" :rows="rows" row-key="name" :actions="false" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.dns-benchmark.note') }}
    </p>
    <div v-if="summary" class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.rank}\t${r.name}\t${r.median} ms\t${r.success}`).join('\n')" :json="summary" filename="dns-benchmark.json" />
    </div>
  </CCard>
</template>
