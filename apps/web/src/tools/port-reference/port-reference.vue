<script setup lang="ts">
import type { PortEntry } from './port-reference.service'
import { NRadioButton, NRadioGroup, NSpin } from 'naive-ui'
import { computed, onMounted, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { loadPorts, MAX_RESULTS, searchPorts } from './port-reference.service'

const { t } = useI18n()
const query = useQueryParam({ name: 'q', defaultValue: '443' })
const protocol = useQueryParam({ name: 'proto', defaultValue: 'all' })
const entries = shallowRef<PortEntry[]>()
onMounted(async () => {
  entries.value = await loadPorts()
})
const result = computed(() => (entries.value ? searchPorts(entries.value, query.value, protocol.value) : undefined))
const rows = computed(() => (result.value?.rows ?? []).map(r => ({ ...r, url: r.reference.startsWith('RFC ') ? `https://www.rfc-editor.org/rfc/rfc${r.reference.slice(4)}` : '' })))
const k = (key: string) => t(`tools.port-reference.columns.${key}`)
const columns = computed(() => [
  { key: 'port', title: k('port'), width: 100, mono: true },
  { key: 'protocol', title: k('protocol'), width: 90 },
  { key: 'service', title: k('service'), width: 160, mono: true },
  { key: 'description', title: k('description'), width: 320 },
  { key: 'reference', title: k('reference'), width: 110, hrefKey: 'url' },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="query" :label="t('tools.port-reference.search')" placeholder="443, ssh, kerberos…" test-id="input" />
    <NRadioGroup v-model:value="protocol" name="protocol" size="small">
      <NRadioButton v-for="p in ['all', 'tcp', 'udp', 'sctp', 'dccp']" :key="p" :value="p" :label="p === 'all' ? t('tools.port-reference.all') : p.toUpperCase()" />
    </NRadioGroup>
  </CCard>
  <NSpin v-if="!entries" class="w-full" />
  <CCard v-else-if="result && result.total" :title="t('tools.port-reference.results', { count: result.total })" data-testid="result">
    <p v-if="result.total > MAX_RESULTS" class="mt-0 text-sm nt-muted">
      {{ t('tools.port-reference.truncated', { max: MAX_RESULTS }) }}
    </p>
    <ResultTable :columns="columns" :rows="rows" filename="ports.json" />
  </CCard>
  <p v-else-if="query.trim()" class="nt-muted" data-testid="no-result">
    {{ t('tools.port-reference.none') }}
  </p>
  <p class="text-xs nt-muted">
    {{ t('tools.port-reference.source') }}
  </p>
</template>
