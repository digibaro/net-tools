<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { convert, MAX_ROWS } from './cidr-range-converter.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'q', defaultValue: '192.168.0.10 - 192.168.0.20' })
const validation = useValidation({
  source: input,
  rules: [{ validator: (v: string) => !!convert(v), message: t('tools.cidr-range-converter.invalid') }],
})
const result = computed(() => convert(input.value))
const k = (key: string) => t(`tools.cidr-range-converter.columns.${key}`)
const columns = computed(() => [
  { key: 'cidr', title: k('cidr'), mono: true },
  { key: 'first', title: k('first'), mono: true },
  { key: 'last', title: k('last'), mono: true },
  { key: 'addresses', title: k('addresses') },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.cidr-range-converter.input')" placeholder="10.0.0.0/22  or  10.0.0.5 - 10.0.1.9" :validation="validation" monospace test-id="input" />
  </CCard>
  <CCard v-if="result" :title="t(`tools.cidr-range-converter.title-${result.mode}`)" data-testid="result">
    <p v-if="result.mode === 'range'" class="mt-0">
      {{ t('tools.cidr-range-converter.summary', { blocks: result.rows.length, total: result.total }) }}
      <span v-if="result.rows.length >= MAX_ROWS" class="nt-muted">{{ t('tools.cidr-range-converter.truncated', { max: MAX_ROWS }) }}</span>
    </p>
    <ResultTable :columns="columns" :rows="result.rows" row-key="cidr" filename="cidr-range.json" />
  </CCard>
</template>
