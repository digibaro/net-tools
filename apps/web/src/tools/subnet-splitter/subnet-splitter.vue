<script setup lang="ts">
import type { SplitMode } from './subnet-splitter.service'
import { NFormItem, NInputNumber, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { MAX_ROWS, split } from './subnet-splitter.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'net', defaultValue: '192.168.0.0/24' })
const mode = useQueryParam<string>({ name: 'mode', defaultValue: 'count' })
const value = useQueryParam({ name: 'n', defaultValue: 4 })
const splitMode = computed(() => (mode.value === 'prefix' ? 'prefix' : 'count') as SplitMode)

const result = computed(() => split(input.value, splitMode.value, value.value))
const validation = useValidation({
  source: input,
  rules: [{ validator: (v: string) => !!split(v, 'count', 1), message: t('tools.subnet-splitter.invalid') }],
})
const k = (key: string) => t(`tools.subnet-splitter.columns.${key}`)
const columns = computed(() => [
  { key: 'index', title: '#', width: 70 },
  { key: 'cidr', title: k('cidr'), mono: true },
  { key: 'first', title: k('first'), mono: true },
  { key: 'last', title: k('last'), mono: true },
  { key: 'addresses', title: k('addresses') },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.subnet-splitter.input')" placeholder="10.0.0.0/16" :validation="validation" monospace test-id="input" />
    <div class="flex flex-wrap items-end gap-4">
      <NFormItem :label="t('tools.subnet-splitter.mode')" :show-feedback="false">
        <NRadioGroup v-model:value="mode" name="split-mode">
          <NRadioButton value="count" :label="t('tools.subnet-splitter.byCount')" />
          <NRadioButton value="prefix" :label="t('tools.subnet-splitter.byPrefix')" />
        </NRadioGroup>
      </NFormItem>
      <NFormItem :label="t(splitMode === 'count' ? 'tools.subnet-splitter.count' : 'tools.subnet-splitter.newPrefix')" :show-feedback="false">
        <NInputNumber v-model:value="value" :min="splitMode === 'count' ? 1 : 0" :max="splitMode === 'count' ? 1048576 : 128" class="w-40" :input-props="{ 'aria-label': t('tools.subnet-splitter.value') }" data-testid="value" />
      </NFormItem>
    </div>
  </CCard>
  <CCard v-if="result" :title="t('tools.subnet-splitter.result', { count: result.total, prefix: result.newPrefix })" data-testid="result">
    <p v-if="result.truncated" class="mt-0 text-sm nt-muted">
      {{ t('tools.subnet-splitter.truncated', { max: MAX_ROWS }) }}
    </p>
    <ResultTable :columns="columns" :rows="result.rows" row-key="cidr" :max-height="480" filename="subnets.json" />
  </CCard>
  <p v-else-if="validation.isValid" class="nt-muted" data-testid="no-result">
    {{ t('tools.subnet-splitter.impossible') }}
  </p>
</template>
