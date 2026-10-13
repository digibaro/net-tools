<script setup lang="ts">
import type { components } from '@/api/schema'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import LatencyChart from './LatencyChart.vue'
import ResultActions from './ResultActions.vue'
import StreamConsole from './StreamConsole.vue'

type Reply = components['schemas']['PingReply']
type Summary = components['schemas']['PingSummary']
const props = defineProps<{ replies: Reply[], summary?: Summary, kind: 'icmp' | 'tcp' }>()
const { t } = useI18n()

const samples = computed(() => props.replies.map(r => (r.error ? null : r.rttMs ?? null)))
const lines = computed(() => props.replies.map(r => (r.error
  ? `seq=${r.seq} ${r.from}: ${r.error}`
  : props.kind === 'icmp'
    ? `${r.bytes ?? ''} bytes from ${r.from}: seq=${r.seq} ttl=${r.ttl} time=${r.rttMs} ms`
    : `connected to ${r.from}: seq=${r.seq} time=${r.rttMs} ms`)))
const rows = computed(() => {
  const s = props.summary
  if (!s)
    return []
  return [
    { label: t('ping.sentReceived'), value: `${s.sent} / ${s.received}`, noCopy: true },
    { label: t('ping.loss'), value: `${s.lossPct} %`, noCopy: true },
    { label: t('ping.rtt'), value: `${s.minMs} / ${s.avgMs} / ${s.maxMs} ms`, noCopy: true },
    { label: t('ping.stddev'), value: `${s.stddevMs} ms`, noCopy: true },
    { label: t('ping.jitter'), value: `${s.jitterMs} ms`, noCopy: true },
  ]
})
const asText = () => [...lines.value, '', ...rows.value.map(r => `${r.label}: ${r.value}`)].join('\n')
</script>

<template>
  <div>
    <LatencyChart v-if="samples.length" :samples="samples" :label="t('ping.chart')" class="mb-3" />
    <StreamConsole :lines="lines" max-height="16rem" />
    <div v-if="summary" class="mt-4" data-testid="summary">
      <CKeyValueList :rows="rows" />
      <div class="mt-3">
        <ResultActions :text="asText" :json="{ replies, summary }" :filename="`${kind}-ping.json`" />
      </div>
    </div>
  </div>
</template>
