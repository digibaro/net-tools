<script setup lang="ts">
import type { EchoInfo } from '@/api/client'
import { NAlert, NButton, NFormItem, NInputNumber, NProgress, NTag } from 'naive-ui'
import { computed, onBeforeUnmount, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getEcho } from '@/api/client'
import LatencyChart from '@/components/LatencyChart.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { MAX_SAMPLES, MIN_INTERVAL, probe, quality, stats } from './browser-latency.service'

const { t } = useI18n()
const count = useQueryParam({ name: 'count', defaultValue: 20 })
const interval = useQueryParam({ name: 'interval', defaultValue: 500 })
const samples = ref<(number | null)[]>([])
const running = ref(false)
const echo = shallowRef<EchoInfo>()
let controller: AbortController | undefined

function sleep(ms: number, signal: AbortSignal) {
  return new Promise<void>((resolve, reject) => {
    const id = setTimeout(resolve, ms)
    signal.addEventListener('abort', () => {
      clearTimeout(id)
      reject(signal.reason)
    }, { once: true })
  })
}

async function start() {
  controller?.abort()
  const c = new AbortController()
  controller = c
  samples.value = []
  running.value = true
  const n = Math.min(MAX_SAMPLES, Math.max(1, count.value))
  const gap = Math.max(MIN_INTERVAL, interval.value)
  try {
    // Warm-up: opens the connection (DNS, TCP, TLS) so samples measure request round trips only.
    echo.value = await getEcho({ signal: c.signal }).catch(() => undefined)
    for (let i = 0; i < n; i++) {
      const started = performance.now()
      samples.value = [...samples.value, await probe('/api/v1/echo', c.signal)]
      if (i < n - 1)
        await sleep(Math.max(0, gap - (performance.now() - started)), c.signal)
    }
  }
  catch {}
  finally {
    if (controller === c)
      running.value = false
  }
}
function stop() {
  controller?.abort()
  running.value = false
}
onBeforeUnmount(stop)

const s = computed(() => stats(samples.value))
const rows = computed(() => {
  const v = s.value
  if (!v)
    return []
  const k = (key: string) => t(`tools.browser-latency.fields.${key}`)
  return [
    { label: t('ping.sentReceived'), value: `${v.sent} / ${v.received}`, noCopy: true },
    { label: t('ping.loss'), value: `${v.lossPct} %`, noCopy: true },
    { label: t('ping.rtt'), value: `${v.min} / ${v.avg} / ${v.max} ms` },
    { label: k('medianP95'), value: `${v.median} / ${v.p95} ms` },
    { label: t('ping.stddev'), value: `${v.stddev} ms`, noCopy: true },
    { label: t('ping.jitter'), value: `${v.jitter} ms`, noCopy: true },
    ...(echo.value ? [{ label: k('protocol'), value: echo.value.protocol, noCopy: true }, { label: k('ip'), value: echo.value.ip, mono: true }] : []),
  ]
})
const qualityType = { excellent: 'success', good: 'success', fair: 'warning', poor: 'error' } as const
const progress = computed(() => Math.round(samples.value.length / Math.max(1, Math.min(MAX_SAMPLES, count.value)) * 100))
</script>

<template>
  <CCard>
    <div class="flex flex-wrap items-end gap-x-6">
      <NFormItem :label="t('ping.count')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="count" :show-button="false" :min="1" :max="MAX_SAMPLES" class="w-32" :input-props="{ 'aria-label': t('ping.count') }" />
      </NFormItem>
      <NFormItem :label="t('ping.interval')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="interval" :show-button="false" :min="MIN_INTERVAL" :max="5000" :step="250" class="w-36" :input-props="{ 'aria-label': t('ping.interval') }" />
      </NFormItem>
      <NButton v-if="!running" type="primary" class="mb-3" data-testid="run" @click="start">
        {{ t('ping.run') }}
      </NButton>
      <NButton v-else class="mb-3" data-testid="stop" @click="stop">
        {{ t('tools.browser-latency.stop') }}
      </NButton>
    </div>
    <NProgress v-if="running" type="line" :percentage="progress" :show-indicator="false" :aria-label="t('tools.browser-latency.progress')" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.browser-latency.note') }}
    </p>
  </CCard>
  <CCard v-if="s" :title="t('common.result')" data-testid="result">
    <template #header-extra>
      <NTag v-if="!running && s.received" :type="qualityType[quality(s)]" :bordered="false" data-testid="quality">
        {{ t(`tools.browser-latency.quality.${quality(s)}`) }}
      </NTag>
    </template>
    <LatencyChart :samples="samples" :label="t('ping.chart')" />
    <NAlert v-if="s.received === 0" type="error" class="my-3">
      {{ t('tools.browser-latency.unreachable') }}
    </NAlert>
    <CKeyValueList class="mt-3" :rows="rows" />
    <div class="mt-3">
      <ResultActions :text="() => samples.map((v, i) => `${i + 1}\t${v === null ? 'lost' : `${v.toFixed(1)} ms`}`).join('\n')" :json="{ stats: s, samples }" filename="browser-latency.json" />
    </div>
  </CCard>
</template>
