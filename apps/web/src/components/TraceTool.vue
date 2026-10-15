<script setup lang="ts">
import type { TraceHop } from '@/utils/trace'
import { NButton, NFormItem, NInputNumber, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, defineAsyncComponent, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import RemoteResults from '@/components/RemoteResults.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import VantageSwitch from '@/components/VantageSwitch.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useRemote } from '@/composables/useRemote'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import { useVantage } from '@/composables/useVantage'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'
import { traceTextReport, upsertHop, withGaps } from '@/utils/trace'

/** Shared page for traceroute (mtr=false) and MTR (mtr=true). */
const props = defineProps<{ mtr: boolean }>()
const { t } = useI18n()
const caps = useCapabilitiesStore()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const proto = useQueryParam({ name: 'proto', defaultValue: 'icmp' })
const port = useQueryParam({ name: 'port', defaultValue: 443 })
const cycles = useQueryParam({ name: 'cycles', defaultValue: 10 })
const maxCycles = computed(() => caps.capabilities?.limits.mtrMaxCycles ?? 20)
const validation = useValidation({ source: target, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })

const hops = ref<TraceHop[]>([])
// Map and its land data load only for finished traces.
const TraceMap = defineAsyncComponent(() => import('./TraceMap.vue'))
const geoAvailable = computed(() => caps.capabilities?.tools['rep.geo']?.enabled === true)
const finished = ref(false)
const { status, error, start, stop } = useSse(props.mtr ? '/v1/mtr/stream' : '/v1/traceroute/stream', {
  hop: (d) => {
    hops.value = upsertHop(hops.value, d as TraceHop)
  },
  done: (d) => {
    hops.value = (d as { summary: { hops: TraceHop[] } }).summary.hops
    finished.value = true
  },
})
const { vantage, locations, limit, available, worldwide } = useVantage()
const remote = useRemote()
const running = computed(() => status.value === 'running' || remote.status.value === 'running')
function stopAll() {
  stop()
  remote.stop()
}
function run() {
  if (!target.value || !validation.value.isValid)
    return
  if (worldwide.value) {
    remote.start({ type: props.mtr ? 'mtr' : 'traceroute', target: target.value, protocol: proto.value.toUpperCase(), locations: locations.value, limit: limit.value })
    return
  }
  hops.value = []
  finished.value = false
  start({ target: target.value, proto: proto.value, port: proto.value === 'tcp' ? port.value : undefined, cycles: props.mtr ? Math.min(cycles.value, maxCycles.value) : undefined })
}
onMounted(() => {
  if (target.value)
    run()
})

const rows = computed(() => withGaps(hops.value).map(h => ({
  hop: h.hop,
  host: h.ip ? (h.host && h.host !== h.ip ? `${h.host}\n${h.ip}` : h.ip) : '*',
  asn: h.asn ? `AS${h.asn}` : '',
  loss: `${h.loss}%`,
  sent: h.sent,
  last: h.lastMs || '—',
  avg: h.avgMs || '—',
  best: h.bestMs || '—',
  worst: h.worstMs || '—',
  stddev: h.stddevMs || '—',
  rtts: h.rttMs.map(r => `${r} ms`).join('  ') || '* * *',
})))
const k = (key: string) => t(`trace.columns.${key}`)
const columns = computed(() => props.mtr
  ? [
      { key: 'hop', title: '#', width: 50 },
      { key: 'host', title: k('host'), mono: true, width: 260 },
      { key: 'asn', title: k('asn'), width: 90 },
      { key: 'loss', title: k('loss'), width: 70 },
      { key: 'sent', title: k('sent'), width: 60 },
      { key: 'last', title: k('last'), width: 70 },
      { key: 'avg', title: k('avg'), width: 70 },
      { key: 'best', title: k('best'), width: 70 },
      { key: 'worst', title: k('worst'), width: 70 },
      { key: 'stddev', title: k('stddev'), width: 70 },
    ]
  : [
      { key: 'hop', title: '#', width: 50 },
      { key: 'host', title: k('host'), mono: true, width: 280 },
      { key: 'asn', title: k('asn'), width: 90 },
      { key: 'rtts', title: k('rtts'), mono: true, width: 240 },
    ])
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="target" :label="t('ping.target')" placeholder="example.com" :validation="validation" monospace test-id="target" />
      <VantageSwitch v-model:vantage="vantage" v-model:locations="locations" v-model:limit="limit" :available="available" />
      <div class="flex flex-wrap items-end gap-x-6">
        <NFormItem :label="t('trace.protocol')" :show-feedback="false" class="mb-3">
          <NRadioGroup v-model:value="proto" name="proto">
            <NRadioButton value="icmp" label="ICMP" />
            <NRadioButton value="udp" label="UDP" />
            <NRadioButton value="tcp" label="TCP" />
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="proto === 'tcp' && !worldwide" :label="t('tools.tcp-ping.port')" :show-feedback="false" class="mb-3 w-28">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.tcp-ping.port') }" />
        </NFormItem>
        <NFormItem v-if="mtr && !worldwide" :label="t('trace.cycles')" :show-feedback="false" class="mb-3 w-28">
          <NInputNumber v-model:value="cycles" :min="1" :max="maxCycles" :show-button="false" :input-props="{ 'aria-label': t('trace.cycles') }" />
        </NFormItem>
      </div>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="!target || !validation.isValid || running" data-testid="run">
          {{ t('ping.run') }}
        </NButton>
        <NButton v-if="running" data-testid="stop" @click="stopAll">
          {{ t('common.stop') }}
        </NButton>
      </div>
    </form>
  </CCard>
  <RemoteResults v-if="worldwide" :meta="remote.meta.value" :probes="remote.probes.value" :summary="remote.summary.value" :status="remote.status.value" :error="remote.error.value" />
  <ApiError v-else-if="error" :error="error" />
  <CCard v-if="!worldwide && rows.length" :title="t('trace.hops', { count: rows.length })" data-testid="result">
    <ResultTable :columns="columns" :rows="rows" :actions="false" class="whitespace-pre-line" :label="t('trace.hops', { count: rows.length })" />
    <TraceMap v-if="finished" :hops="hops" :geo="geoAvailable" />
    <div v-if="finished" class="mt-3" data-testid="finished">
      <ResultActions :text="() => traceTextReport(hops, mtr)" :json="hops" :filename="mtr ? 'mtr.json' : 'traceroute.json'" />
    </div>
  </CCard>
</template>
