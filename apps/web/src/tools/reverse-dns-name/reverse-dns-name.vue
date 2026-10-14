<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { reverseNames } from './reverse-dns-name.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'q', defaultValue: '192.0.2.10\n198.51.100.0/24\n203.0.113.64/26\n2001:db8::1\n2001:db8:abcd::/48' })
const result = computed(() => reverseNames(input.value))
const validation = computed(() => result.value.invalid.length
  ? { isValid: false, status: 'error' as const, message: t('tools.reverse-dns-name.invalid', { count: result.value.invalid.length, entries: result.value.invalid.slice(0, 5).join(', ') }) }
  : undefined)

const rows = computed(() => result.value.rows.map(r => ({
  input: r.input,
  kind: t(`tools.reverse-dns-name.kinds.${r.kind}`),
  names: r.names.join('\n'),
  note: r.kind === 'classless'
    ? t('tools.reverse-dns-name.classlessNote', { parent: r.note })
    : r.note ? t('tools.reverse-dns-name.truncated', { total: r.note }) : r.names.length > 1 ? t('tools.reverse-dns-name.multiple', { count: r.names.length }) : '',
})))
const k = (key: string) => t(`tools.reverse-dns-name.columns.${key}`)
const columns = computed(() => [
  { key: 'input', title: k('input'), mono: true },
  { key: 'kind', title: k('kind'), width: 110 },
  { key: 'names', title: k('names'), mono: true, width: 380 },
  { key: 'note', title: k('note'), width: 220 },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.reverse-dns-name.input')" :validation="validation" multiline :rows="6" monospace test-id="input" />
  </CCard>
  <CCard v-if="rows.length" :title="t('common.result')" data-testid="result">
    <ResultTable :columns="columns" :rows="rows" filename="reverse-dns-names.json" class="whitespace-pre-line" />
  </CCard>
</template>
