<script setup lang="ts">
import type { OpenPort, PortscanSummary } from './port-scanner.service'
import { NAlert, NButton, NProgress, NTag } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'
import { isPortSpec, PRESETS } from './port-scanner.service'

const { t } = useI18n()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const ports = useQueryParam({ name: 'ports', defaultValue: 'top100' })
const targetV = useValidation({ source: target, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const portsV = useValidation({ source: ports, rules: [{ validator: isPortSpec, message: t('tools.port-scanner.invalidPorts') }] })
const open = ref<OpenPort[]>([])
const progress = ref({ done: 0, total: 0 })
const summary = shallowRef<PortscanSummary>()
const { status, error, start, stop } = useSse('/v1/portscan/stream', {
  meta: d => (progress.value = { done: 0, total: (d as { ports: number }).ports }),
  port: d => open.value.push(d as OpenPort),
  progress: d => (progress.value = d as { done: number, total: number }),
  done: d => (summary.value = (d as { summary: PortscanSummary }).summary),
})
const running = computed(() => status.value === 'running')
const canRun = computed(() => !!target.value && targetV.value.isValid && portsV.value.isValid)
function run() {
  if (!canRun.value)
    return
  open.value = []
  summary.value = undefined
  start({ target: target.value.trim(), ports: ports.value.trim() })
}
onMounted(() => {
  if (target.value)
    run()
})
const rows = computed(() => [...open.value].sort((a, b) => a.port - b.port).map(p => ({ port: p.port, service: p.service || '—', link: `/port-reference?q=${p.port}` })))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_14rem]">
        <CInputText v-model:value="target" :label="t('ping.target')" placeholder="scanme.example.com" :validation="targetV" monospace test-id="target" />
        <CInputText v-model:value="ports" :label="t('tools.port-scanner.ports')" placeholder="top100" :validation="portsV" monospace test-id="ports" />
      </div>
      <div class="mb-3 flex flex-wrap gap-1">
        <NTag v-for="p in PRESETS" :key="p" size="small" class="cursor-pointer" :bordered="false" :type="ports === p ? 'primary' : 'default'" @click="ports = p">
          {{ t(`tools.port-scanner.presets.${p}`) }}
        </NTag>
      </div>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="!canRun || running" data-testid="run">
          {{ t('tools.port-scanner.run') }}
        </NButton>
        <NButton v-if="running" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.port-scanner.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="running || summary" :title="t('common.result')" data-testid="result">
    <NProgress v-if="running" type="line" :percentage="progress.total ? Math.round(progress.done / progress.total * 100) : 0" :show-indicator="false" :aria-label="t('tools.port-scanner.run')" class="mb-3" />
    <NAlert v-if="summary" :type="summary.open.length ? 'info' : 'success'" class="mb-3">
      {{ t('tools.port-scanner.summary', { open: summary.open.length, closed: summary.closed, filtered: summary.filtered, address: summary.address }) }}
    </NAlert>
    <ResultTable v-if="rows.length" :columns="[{ key: 'port', title: t('tools.port-scanner.columns.port'), width: 90, routeKey: 'link' }, { key: 'service', title: t('tools.port-scanner.columns.service'), width: 180 }]" :rows="rows" row-key="port" :actions="false" />
    <div v-if="summary" class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.port}/tcp open ${r.service}`).join('\n')" :json="summary" filename="portscan.json" />
    </div>
  </CCard>
</template>
