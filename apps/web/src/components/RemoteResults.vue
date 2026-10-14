<script setup lang="ts">
import type { ApiError as ApiErrorType } from '@/api/client'
import type { RemoteMeta, RemoteProbeResult, RemoteSummary } from '@/composables/useRemote'
import type { SseStatus } from '@/composables/useSse'
import { NProgress, NTag } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import LatencyChart from '@/components/LatencyChart.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { flag } from '@/tools/ip-geolocation/ip-geolocation.service'
import CCard from '@/ui/c-card.vue'
import { countryName, hopRows, pingSummary, verdict } from './remote'

/** Globalping results, one block per probe; layout adapts to the measurement type. */
const props = defineProps<{ meta?: RemoteMeta, probes: RemoteProbeResult[], summary?: RemoteSummary, status: SseStatus, error?: ApiErrorType }>()
const { t, locale } = useI18n()
const type = computed(() => props.meta?.type ?? 'ping')
const expected = computed(() => props.meta?.probes ?? 0)
const up = computed(() => props.probes.filter(p => verdict(type.value, p.result) === 'up').length)
const verdictType = { up: 'success', down: 'error', error: 'warning' } as const
const where = (p: RemoteProbeResult) => [p.probe.city, countryName(p.probe.country, locale.value)].filter(Boolean).join(', ')
const rtts = (p: RemoteProbeResult) => ((p.result.timings ?? []) as { rtt?: number | null }[]).map(x => (typeof x.rtt === 'number' ? x.rtt : null))
const httpTimings = (p: RemoteProbeResult) => (p.result.timings ?? {}) as Record<string, number | null>
const c = (key: string) => t(`remote.columns.${key}`)
const asText = () => props.probes.map(p => `${where(p)} (AS${p.probe.asn} ${p.probe.network}): ${verdict(type.value, p.result)} ${type.value === 'ping' ? pingSummary(p.result) : ''}`).join('\n')
</script>

<template>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="meta" :title="t('remote.title', { target: meta.target })" data-testid="remote-result">
    <p class="mt-0 text-sm nt-muted">
      {{ t('remote.answered', { up, done: probes.length, total: expected }) }}
      <span v-if="summary && !summary.finished"> · {{ t('remote.incomplete') }}</span>
    </p>
    <NProgress v-if="status === 'running'" type="line" :percentage="expected ? Math.round(probes.length / expected * 100) : 0" :show-indicator="false" :aria-label="t('remote.progress')" class="mb-3" />
    <div v-for="p in probes" :key="p.index" class="mb-3 border nt-border rounded border-solid p-3" data-testid="remote-probe">
      <div class="flex flex-wrap items-center gap-2">
        <span aria-hidden="true">{{ flag(p.probe.country) }}</span>
        <span class="font-semibold">{{ where(p) }}</span>
        <span class="text-xs nt-muted">AS{{ p.probe.asn }} {{ p.probe.network }}</span>
        <NTag size="small" :bordered="false" :type="verdictType[verdict(type, p.result)]" class="ml-auto">
          {{ t(`remote.verdict.${verdict(type, p.result)}`) }}
        </NTag>
      </div>
      <template v-if="type === 'ping'">
        <p class="m-0 mt-2 text-sm font-mono">
          {{ pingSummary(p.result) }}
        </p>
        <LatencyChart :samples="rtts(p)" :height="60" :label="t('ping.chart')" />
      </template>
      <div v-else-if="type === 'traceroute' || type === 'mtr'" class="mt-2">
        <ResultTable :columns="[{ key: 'hop', title: '#', width: 50 }, { key: 'ip', title: c('ip'), mono: true, width: 170 }, { key: 'host', title: c('host'), mono: true, width: 220 }, { key: 'asn', title: 'ASN', width: 100 }, { key: 'avg', title: c('avg'), width: 90 }, ...(type === 'mtr' ? [{ key: 'loss', title: c('loss'), width: 80 }] : [])]" :rows="hopRows(p.result)" row-key="hop" :filename="`remote-${p.index}.json`" />
      </div>
      <div v-else-if="type === 'dns'" class="mt-2 text-sm">
        <p class="m-0">
          {{ p.result.statusCodeName }} · {{ httpTimings(p).total }} ms · {{ t('remote.resolver') }} {{ p.result.resolver }}
        </p>
        <p v-for="(a, i) in p.result.answers ?? []" :key="i" class="m-0 break-all font-mono">
          {{ a.name }} {{ a.ttl }} {{ a.type }} {{ a.value }}
        </p>
      </div>
      <div v-else-if="type === 'http'" class="mt-2 text-sm">
        <p class="m-0">
          HTTP {{ p.result.statusCode }} {{ p.result.statusCodeName }}<template v-if="p.result.tls">
            · {{ p.result.tls.protocol }}
          </template>
        </p>
        <p class="m-0 nt-muted font-mono">
          {{ ['dns', 'tcp', 'tls', 'firstByte', 'total'].map(k => `${c(k)} ${httpTimings(p)[k] ?? '—'} ms`).join(' · ') }}
        </p>
      </div>
      <details v-if="p.result.rawOutput" class="mt-2">
        <summary class="cursor-pointer text-xs">
          {{ t('remote.raw') }}
        </summary>
        <pre class="m-0 mt-1 max-h-60 overflow-auto text-xs font-mono" tabindex="0">{{ p.result.rawOutput }}</pre>
      </details>
    </div>
    <ResultActions v-if="status !== 'running'" :text="asText" :json="{ meta, probes }" filename="globalping.json" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('remote.attribution') }}
    </p>
  </CCard>
</template>
