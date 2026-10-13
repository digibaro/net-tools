<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { convertList } from './punycode-converter.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'q', defaultValue: 'münchen.de\nxn--bcher-kva.example\n例え.テスト' })
const result = computed(() => convertList(input.value))
const validation = computed(() => result.value.invalid.length
  ? { isValid: false, status: 'error' as const, message: t('tools.punycode-converter.invalid', { count: result.value.invalid.length, entries: result.value.invalid.slice(0, 5).join(', ') }) }
  : undefined)
const k = (key: string) => t(`tools.punycode-converter.columns.${key}`)
const columns = computed(() => [
  { key: 'input', title: k('input'), width: 200 },
  { key: 'ascii', title: k('ascii'), mono: true, width: 240 },
  { key: 'unicode', title: k('unicode'), width: 200 },
  { key: 'idnLabels', title: k('idnLabels'), width: 120 },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.punycode-converter.input')" :validation="validation" multiline :rows="5" test-id="input" />
  </CCard>
  <CCard v-if="result.rows.length" :title="t('common.result')" data-testid="result">
    <ResultTable :columns="columns" :rows="result.rows" filename="idn.json" />
  </CCard>
</template>
