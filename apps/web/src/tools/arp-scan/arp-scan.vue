<script setup lang="ts">
import type { OuiTable } from '../mac-address-lookup/mac-address-lookup.service'
import type { ArpHost, ArpScanSummary } from './arp-scan.service'
import { NAlert, NButton } from 'naive-ui'
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
import { loadOuiTable, lookup } from '../mac-address-lookup/mac-address-lookup.service'
import { isScannable } from './arp-scan.service'

const { t } = useI18n()
const cidr = useQueryParam({ name: 'cidr', defaultValue: '192.168.1.0/24' })
const validation = useValidation({ source: cidr, rules: [{ validator: isScannable, message: t('tools.arp-scan.invalid') }] })
const hosts = ref<ArpHost[]>([])
const summary = shallowRef<ArpScanSummary>()
const oui = shallowRef<OuiTable>()
const { status, error, start } = useSse('/v1/lan/arpscan/stream', {
  host: d => hosts.value.push(d as ArpHost),
  done: d => (summary.value = (d as { summary: ArpScanSummary }).summary),
})
const running = computed(() => status.value === 'running')
async function run() {
  if (!validation.value.isValid)
    return
  hosts.value = []
  summary.value = undefined
  start({ cidr: cidr.value.trim() })
  oui.value ??= await loadOuiTable() // vendor names come from the bundled OUI registry
}
onMounted(() => {
  if (new URLSearchParams(location.search).has('cidr'))
    void run()
})
const rows = computed(() => [...hosts.value]
  .sort((a, b) => a.ip.localeCompare(b.ip, undefined, { numeric: true }))
  .map(h => ({ ip: h.ip, mac: h.mac, vendor: (oui.value && lookup(h.mac, oui.value)?.vendor) || '—' })))
const c = (k: string) => t(`tools.arp-scan.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="cidr" :label="t('tools.arp-scan.subnet')" placeholder="192.168.1.0/24" :validation="validation" monospace test-id="cidr" />
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!validation.isValid || running" data-testid="run">
        {{ t('tools.arp-scan.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.arp-scan.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="hosts.length || summary" :title="t('tools.arp-scan.found', { n: hosts.length })" data-testid="result">
    <NAlert v-if="summary" type="info" class="mb-3">
      {{ t('tools.arp-scan.summary', { n: summary.hosts.length, scanned: summary.scanned, subnet: summary.subnet, iface: summary.interface }) }}
    </NAlert>
    <ResultTable :columns="[{ key: 'ip', title: c('ip'), mono: true, width: 150 }, { key: 'mac', title: c('mac'), mono: true, width: 170 }, { key: 'vendor', title: c('vendor'), width: 260 }]" :rows="rows" row-key="ip" :actions="false" />
    <div v-if="summary" class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.ip}\t${r.mac}\t${r.vendor}`).join('\n')" :json="rows" filename="arp-scan.json" />
    </div>
  </CCard>
</template>
