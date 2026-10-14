<script setup lang="ts">
import type { Quantity } from './data-unit-converter.service'
import { NFormItem, NInputNumber, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CSelect from '@/ui/c-select.vue'
import { convertAll, unitsFor } from './data-unit-converter.service'

const { t } = useI18n()
const quantityParam = useQueryParam({ name: 'kind', defaultValue: 'size' })
const quantity = computed<Quantity>({
  get: () => (quantityParam.value === 'rate' ? 'rate' : 'size'),
  set: (v) => {
    quantityParam.value = v
  },
})
const value = useQueryParam({ name: 'value', defaultValue: 1 })
const unit = useQueryParam({ name: 'unit', defaultValue: 'GiB' })
const options = computed(() => unitsFor(quantity.value).map(u => ({ label: u.label, value: u.id })))
watch(quantity, (q) => {
  if (!unitsFor(q).some(u => u.id === unit.value))
    unit.value = q === 'rate' ? 'Mbps' : 'GiB'
})
const rows = computed(() => convertAll(value.value, unit.value, quantity.value) ?? [])
const columns = computed(() => [
  { key: 'unit', title: t('tools.data-unit-converter.unit'), width: 140 },
  { key: 'value', title: t('tools.data-unit-converter.value'), mono: true, width: 260 },
])
</script>

<template>
  <CCard>
    <NRadioGroup v-model:value="quantity" name="quantity" class="mb-3">
      <NRadioButton value="size" :label="t('tools.data-unit-converter.size')" />
      <NRadioButton value="rate" :label="t('tools.data-unit-converter.rate')" />
    </NRadioGroup>
    <div class="flex gap-2">
      <NFormItem :label="t('tools.data-unit-converter.value')" :show-feedback="false" class="mb-3 flex-1">
        <NInputNumber v-model:value="value" :show-button="false" :input-props="{ 'aria-label': t('tools.data-unit-converter.value') }" data-testid="value" />
      </NFormItem>
      <div class="w-36">
        <CSelect v-model:value="unit" :label="t('tools.data-unit-converter.unit')" :options="options" />
      </div>
    </div>
  </CCard>
  <CCard v-if="rows.length" :title="t('common.result')" data-testid="result">
    <ResultTable :columns="columns" :rows="rows.map(r => ({ unit: r.unit, value: r.value }))" filename="units.json" />
  </CCard>
</template>
