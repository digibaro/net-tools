<script setup lang="ts">
import type { LatencyRow } from './latency-matrix.service'
import { NButton, NFormItem, NInputNumber, NProgress } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import LatencyChart from '@/components/LatencyChart.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'
import { parseTargets, ranked } from './latency-matrix.service'

const { t } = useI18n()
const caps = useCapabilitiesStore()
const targets = useQueryParam({ name: 'targets', defaultValue: '1.1.1.1, 8.8.8.8, 9.9.9.9' })
const count = useQueryParam({ name: 'count', defaultValue: 5 })
const maxCount = computed(() => caps.capabilities?.limits.pingMaxCount ?? 20)
const list = computed(() => parseTargets(targets.value))
const invalid = computed(() => list.value.filter(x => !isValidHost(x)))
const validation = computed(() => (list.value.length > 10
  ? { isValid: false, status: 'error' as const, message: t('tools.latency-matrix.tooMany') }
  : invalid.value.length ? { isValid: false, status: 'error' as const, message: t('tools.latency-matrix.invalid', { list: invalid.value.join(', ') }) } : { isValid: true, message: '', status: undefined }))
const rows = ref<LatencyRow[]>([])
const { status, error, start, stop } = useSse('/v1/latency-matrix/stream', { row: d => rows.value.push(d as LatencyRow) })
const running = computed(() => status.value === 'running')
function run() {
  if (!list.value.length || !validation.value.isValid)
    return
  rows.value = []
  start({ targets: list.value.join(','), count: Math.min(count.value, maxCount.value) })
}
onMounted(() => {
  if (targets.value && new URLSearchParams(location.search).has('targets'))
    run()
})
const sorted = computed(() => ranked(rows.value))
const best = computed(() => sorted.value.find(r => r.stats.received)?.stats.avgMs ?? 0)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="targets" :label="t('tools.latency-matrix.targets')" placeholder="1.1.1.1, example.com" :validation="validation" multiline :rows="2" monospace test-id="targets" />
      <NFormItem :label="t('ping.count')" :show-feedback="false" class="mb-3 w-32">
        <NInputNumber v-model:value="count" :min="1" :max="maxCount" :show-button="false" :input-props="{ 'aria-label': t('ping.count') }" />
      </NFormItem>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="!list.length || !validation.isValid || running" data-testid="run">
          {{ t('ping.run') }}
        </NButton>
        <NButton v-if="running" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="rows.length" :title="t('common.result')" data-testid="result">
    <div v-for="(r, i) in sorted" :key="r.target" class="mb-3 border-b nt-border border-b-solid pb-3 last:border-b-0">
      <div class="flex flex-wrap items-baseline gap-2">
        <span class="font-semibold">{{ r.stats.received ? i + 1 : '—' }}.</span>
        <span class="font-mono">{{ r.target }}</span>
        <span v-if="r.address && r.address !== r.target" class="text-xs nt-muted font-mono">{{ r.address }}</span>
        <span class="ml-auto text-sm font-mono">
          <template v-if="r.error">{{ r.error }}</template>
          <template v-else-if="r.stats.received">{{ r.stats.minMs }} / {{ r.stats.avgMs }} / {{ r.stats.maxMs }} ms · {{ r.stats.lossPct }} %</template>
          <template v-else>{{ t('tools.latency-matrix.noReply') }}</template>
        </span>
      </div>
      <NProgress v-if="r.stats.received" type="line" :percentage="Math.min(100, Math.round(best / Math.max(r.stats.avgMs, 0.001) * 100))" :show-indicator="false" :aria-label="r.target" class="my-1" />
      <LatencyChart v-if="r.stats.received" :samples="r.rtts" :height="50" :label="t('ping.chart')" />
    </div>
    <ResultActions v-if="!running" :text="() => sorted.map(r => `${r.target}\t${r.error || `${r.stats.avgMs} ms avg, ${r.stats.lossPct} % loss`}`).join('\n')" :json="sorted" filename="latency-matrix.json" />
  </CCard>
</template>
