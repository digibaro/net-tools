<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { SPECIAL_RANGES } from '@/utils/special-ranges'
import { checkIps } from './ip-special-check.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'ips', defaultValue: '10.20.30.40\n100.64.1.1\n192.0.2.10\n8.8.8.8\nfd00::1\n2001:db8::1' })
const result = computed(() => checkIps(input.value))
const validation = computed(() => result.value.invalid.length
  ? { isValid: false, status: 'error' as const, message: t('tools.ip-special-check.invalid', { count: result.value.invalid.length, entries: result.value.invalid.slice(0, 5).join(', ') }) }
  : undefined)

const yesNo = (b: boolean) => t(b ? 'common.yes' : 'common.no')
const rows = computed(() => result.value.rows.map(r => ({ ...r, globallyReachable: yesNo(r.globallyReachable), bogon: yesNo(r.bogon) })))
const k = (key: string) => t(`tools.ip-special-check.columns.${key}`)
const columns = computed(() => [
  { key: 'ip', title: k('ip'), mono: true, sortable: true },
  { key: 'name', title: k('name'), sortable: true },
  { key: 'range', title: k('range'), mono: true },
  { key: 'rfc', title: k('rfc') },
  { key: 'bogon', title: k('bogon'), sortable: true },
])
const registry = computed(() => SPECIAL_RANGES.map(r => ({ ...r, globallyReachable: yesNo(r.globallyReachable) })))
const registryColumns = computed(() => [
  { key: 'cidr', title: k('range'), mono: true, sortable: true },
  { key: 'name', title: k('name'), sortable: true },
  { key: 'rfc', title: k('rfc') },
  { key: 'globallyReachable', title: k('globallyReachable') },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.ip-special-check.input')" :validation="validation" multiline :rows="6" monospace test-id="input" />
  </CCard>
  <CCard v-if="rows.length" :title="t('common.result')" data-testid="result">
    <ResultTable :columns="columns" :rows="rows" filename="ip-special-check.json" />
  </CCard>
  <CCard :title="t('tools.ip-special-check.registry')">
    <p class="mt-0 text-sm nt-muted">
      {{ t('tools.ip-special-check.registryHint') }}
    </p>
    <ResultTable :columns="registryColumns" :rows="registry" :actions="false" :max-height="400" :label="t('tools.ip-special-check.registry')" />
  </CCard>
</template>
