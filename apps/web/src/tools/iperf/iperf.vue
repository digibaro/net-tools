<script setup lang="ts">
import type { IperfInterval, IperfSummary } from './iperf.service'
import { NButton, NFormItem, NInputNumber, NRadioButton, NRadioGroup, NSwitch } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidHost } from '@/utils/host'
import { formatBytes, formatRate } from './iperf.service'

const { t } = useI18n()
const caps = useCapabilitiesStore()
const maxSeconds = computed(() => caps.capabilities?.limits.iperfMaxSeconds ?? 15)
const server = useQueryParam({ name: 'server', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 5201 })
const proto = useQueryParam({ name: 'proto', defaultValue: 'tcp' })
const duration = useQueryParam({ name: 'duration', defaultValue: 10 })
const reverse = useQueryParam({ name: 'reverse', defaultValue: false })
const parallel = useQueryParam({ name: 'parallel', defaultValue: 1 })
const bitrate = useQueryParam({ name: 'bitrate', defaultValue: 10 })
const serverV = useValidation({ source: server, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const intervals = ref<IperfInterval[]>([])
const summary = shallowRef<IperfSummary>()
const { status, error, start, stop } = useSse('/v1/iperf/stream', {
  interval: d => intervals.value.push(d as IperfInterval),
  done: d => (summary.value = (d as { summary: IperfSummary }).summary),
})
const running = computed(() => status.value === 'running')
const canRun = computed(() => !!server.value && serverV.value.isValid)
function run() {
  if (!canRun.value)
    return
  intervals.value = []
  summary.value = undefined
  start({ server: server.value.trim(), port: port.value, proto: proto.value, duration: Math.min(duration.value, maxSeconds.value), reverse: reverse.value || undefined, parallel: parallel.value, bitrate: proto.value === 'udp' ? bitrate.value : undefined })
}
onMounted(() => {
  if (server.value)
    run()
})
const rows = computed(() => intervals.value.map(i => ({
  id: i.start,
  time: `${i.start.toFixed(1)}–${i.end.toFixed(1)} s`,
  rate: formatRate(i.mbps),
  bytes: formatBytes(i.bytes),
  extra: i.retransmits !== undefined ? String(i.retransmits) : i.packets !== undefined ? String(i.packets) : '—',
})))
const summaryRows = computed(() => {
  const s = summary.value
  if (!s)
    return []
  const out = [
    { label: t('tools.iperf.direction'), value: s.reverse ? t('tools.iperf.download') : t('tools.iperf.upload'), noCopy: true },
    { label: t('tools.iperf.sent'), value: `${formatRate(s.sentMbps)} (${formatBytes(s.sentBytes)})`, mono: true },
    { label: t('tools.iperf.received'), value: formatRate(s.receivedMbps), mono: true },
  ]
  if (s.retransmits !== undefined)
    out.push({ label: t('tools.iperf.retransmits'), value: String(s.retransmits), mono: true })
  if (s.jitterMs !== undefined)
    out.push({ label: t('tools.iperf.jitter'), value: `${s.jitterMs.toFixed(3)} ms`, mono: true })
  if (s.lostPercent !== undefined)
    out.push({ label: t('tools.iperf.loss'), value: `${s.lostPercent.toFixed(2)} % (${s.lostPackets ?? 0})`, mono: true })
  out.push({ label: t('tools.iperf.server'), value: `${s.server} · ${s.version}`, mono: true })
  return out
})
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="server" :label="t('tools.iperf.serverLabel')" placeholder="iperf.example.net" :validation="serverV" monospace test-id="server" />
        <NFormItem :label="t('tools.iperf.port')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.iperf.port') }" />
        </NFormItem>
      </div>
      <div class="flex flex-wrap items-end gap-x-6">
        <NFormItem :label="t('tools.iperf.protocol')" :show-feedback="false" class="mb-3">
          <NRadioGroup v-model:value="proto" name="iperf-proto">
            <NRadioButton value="tcp" label="TCP" />
            <NRadioButton value="udp" label="UDP" />
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('tools.iperf.duration')" :show-feedback="false" class="mb-3 w-28">
          <NInputNumber v-model:value="duration" :min="1" :max="maxSeconds" :show-button="false" :input-props="{ 'aria-label': t('tools.iperf.duration') }" />
        </NFormItem>
        <NFormItem :label="t('tools.iperf.parallel')" :show-feedback="false" class="mb-3 w-28">
          <NInputNumber v-model:value="parallel" :min="1" :max="4" :show-button="false" :input-props="{ 'aria-label': t('tools.iperf.parallel') }" />
        </NFormItem>
        <NFormItem v-if="proto === 'udp'" :label="t('tools.iperf.bitrate')" :show-feedback="false" class="mb-3 w-32">
          <NInputNumber v-model:value="bitrate" :min="1" :max="1000" :show-button="false" :input-props="{ 'aria-label': t('tools.iperf.bitrate') }" />
        </NFormItem>
        <NFormItem :label="t('tools.iperf.reverse')" :show-feedback="false" class="mb-3">
          <NSwitch v-model:value="reverse" :aria-label="t('tools.iperf.reverse')" />
        </NFormItem>
      </div>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="!canRun || running" data-testid="run">
          {{ t('tools.iperf.run') }}
        </NButton>
        <NButton v-if="running" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.iperf.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="intervals.length || summary" :title="t('common.result')" data-testid="result">
    <CKeyValueList v-if="summary" :rows="summaryRows" class="mb-3" />
    <ResultTable :columns="[{ key: 'time', title: t('tools.iperf.columns.time'), width: 120 }, { key: 'rate', title: t('tools.iperf.columns.rate'), mono: true, width: 140 }, { key: 'bytes', title: t('tools.iperf.columns.bytes'), mono: true, width: 120 }, { key: 'extra', title: proto === 'udp' ? t('tools.iperf.columns.packets') : t('tools.iperf.columns.retransmits'), mono: true, width: 120 }]" :rows="rows" row-key="id" :actions="false" />
    <div v-if="summary" class="mt-3">
      <ResultActions :text="() => summaryRows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="{ summary, intervals }" filename="iperf.json" />
    </div>
  </CCard>
</template>
