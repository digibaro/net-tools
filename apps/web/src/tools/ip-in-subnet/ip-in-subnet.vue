<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { matchIps } from './ip-in-subnet.service'

const { t } = useI18n()
const ips = useQueryParam({ name: 'ips', defaultValue: '10.1.2.3\n192.168.5.20\n8.8.8.8' })
const prefixes = useQueryParam({ name: 'prefixes', defaultValue: '10.0.0.0/8\n172.16.0.0/12\n192.168.0.0/16' })
const result = computed(() => matchIps(ips.value, prefixes.value))

function invalid(list: string[]) {
  return list.length
    ? { isValid: false, status: 'error' as const, message: t('tools.ip-in-subnet.invalid', { count: list.length, entries: list.slice(0, 5).join(', ') }) }
    : undefined
}

const rows = computed(() => result.value.rows.map(r => ({
  ip: r.ip,
  matched: t(r.matched ? 'common.yes' : 'common.no'),
  prefixes: r.prefixes.join(', '),
})))
const k = (key: string) => t(`tools.ip-in-subnet.columns.${key}`)
const columns = computed(() => [
  { key: 'ip', title: k('ip'), mono: true, sortable: true },
  { key: 'matched', title: k('matched'), sortable: true },
  { key: 'prefixes', title: k('prefixes'), mono: true },
])
const matchedCount = computed(() => result.value.rows.filter(r => r.matched).length)
</script>

<template>
  <div class="grid gap-x-4 md:grid-cols-2">
    <CCard>
      <CInputText v-model:value="ips" :label="t('tools.ip-in-subnet.ips')" :validation="invalid(result.invalidIps)" multiline :rows="6" monospace test-id="ips" />
    </CCard>
    <CCard>
      <CInputText v-model:value="prefixes" :label="t('tools.ip-in-subnet.prefixes')" :validation="invalid(result.invalidPrefixes)" multiline :rows="6" monospace test-id="prefixes" />
    </CCard>
  </div>
  <CCard v-if="rows.length" :title="t('tools.ip-in-subnet.result', { matched: matchedCount, total: rows.length })" data-testid="result">
    <ResultTable :columns="columns" :rows="rows" filename="ip-in-subnet.json" />
  </CCard>
</template>
