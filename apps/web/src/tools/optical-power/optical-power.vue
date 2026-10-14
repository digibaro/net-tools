<script setup lang="ts">
import type { PowerUnit } from './optical-power.service'
import { NAlert, NButton, NFormItem, NInputNumber } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { formatNumber } from '@/utils/units'
import { FIBER_PRESETS, fromMilliwatts, linkBudget, POWER_UNITS, toMilliwatts } from './optical-power.service'

const { t } = useI18n()
const value = useQueryParam({ name: 'p', defaultValue: -3 })
const unit = useQueryParam({ name: 'u', defaultValue: 'dBm' })
const unitLabel = (u: string) => (u === 'uW' ? 'µW' : u)
const unitOptions = POWER_UNITS.map(u => ({ label: unitLabel(u), value: u }))
const converted = computed(() => {
  const mw = toMilliwatts(value.value, unit.value as PowerUnit)
  if (mw === undefined)
    return []
  const all = fromMilliwatts(mw)
  return POWER_UNITS.map(u => ({ label: unitLabel(u), value: `${formatNumber(all[u], 6)} ${unitLabel(u)}`, mono: true }))
})

const fields = {
  tx: useQueryParam({ name: 'tx', defaultValue: -3 }),
  sens: useQueryParam({ name: 'sens', defaultValue: -20 }),
  overload: useQueryParam({ name: 'ovl', defaultValue: 0 }),
  km: useQueryParam({ name: 'km', defaultValue: 10 }),
  att: useQueryParam({ name: 'att', defaultValue: 0.35 }),
  conn: useQueryParam({ name: 'conn', defaultValue: 2 }),
  connLoss: useQueryParam({ name: 'cl', defaultValue: 0.5 }),
  splices: useQueryParam({ name: 'spl', defaultValue: 2 }),
  spliceLoss: useQueryParam({ name: 'sl', defaultValue: 0.1 }),
  margin: useQueryParam({ name: 'margin', defaultValue: 3 }),
}
const inputs = [
  { key: 'tx', step: 0.5 },
  { key: 'sens', step: 0.5 },
  { key: 'overload', step: 0.5 },
  { key: 'km', step: 1, min: 0 },
  { key: 'att', step: 0.01, min: 0 },
  { key: 'conn', step: 1, min: 0 },
  { key: 'connLoss', step: 0.05, min: 0 },
  { key: 'splices', step: 1, min: 0 },
  { key: 'spliceLoss', step: 0.01, min: 0 },
  { key: 'margin', step: 0.5, min: 0 },
] as const

const budget = computed(() => linkBudget({
  txDbm: fields.tx.value,
  rxSensitivityDbm: fields.sens.value,
  rxOverloadDbm: fields.overload.value,
  lengthKm: fields.km.value,
  attenuationDbPerKm: fields.att.value,
  connectors: fields.conn.value,
  connectorLossDb: fields.connLoss.value,
  splices: fields.splices.value,
  spliceLossDb: fields.spliceLoss.value,
  marginDb: fields.margin.value,
}))
const db = (n: number) => `${formatNumber(n, 4)} dB`
const budgetRows = computed(() => {
  const b = budget.value
  if (!b)
    return []
  const k = (key: string) => t(`tools.optical-power.results.${key}`)
  return [
    { label: k('fiber'), value: db(b.fiberLoss) },
    { label: k('connectors'), value: db(b.connectorLoss) },
    { label: k('splices'), value: db(b.spliceLoss) },
    { label: k('total'), value: db(b.totalLoss) },
    { label: k('rx'), value: `${formatNumber(b.rxDbm, 4)} dBm` },
    { label: k('budget'), value: db(b.budget) },
    { label: k('headroom'), value: db(b.headroom) },
  ]
})
</script>

<template>
  <CCard :title="t('tools.optical-power.convertTitle')">
    <div class="grid gap-x-4 sm:grid-cols-[1fr_9rem]">
      <NFormItem :label="t('tools.optical-power.power')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="value" :show-button="false" :input-props="{ 'aria-label': t('tools.optical-power.power') }" data-testid="power" />
      </NFormItem>
      <CSelect v-model:value="unit" :label="t('tools.optical-power.unit')" :options="unitOptions" />
    </div>
    <CKeyValueList v-if="converted.length" :rows="converted" data-testid="converted" />
    <p v-else class="m-0 text-sm nt-muted">
      {{ t('tools.optical-power.positive') }}
    </p>
  </CCard>
  <CCard :title="t('tools.optical-power.budgetTitle')">
    <div class="mb-3 flex flex-wrap gap-2">
      <NButton v-for="p in FIBER_PRESETS" :key="p.id" size="small" secondary @click="fields.att.value = p.attenuation">
        {{ t(`tools.optical-power.presets.${p.id}`) }} · {{ p.attenuation }} dB/km
      </NButton>
    </div>
    <div class="grid gap-x-4 lg:grid-cols-3 sm:grid-cols-2">
      <NFormItem v-for="f in inputs" :key="f.key" :label="t(`tools.optical-power.fields.${f.key}`)" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="fields[f.key].value" :show-button="false" :step="f.step" :min="'min' in f ? f.min : undefined" :input-props="{ 'aria-label': t(`tools.optical-power.fields.${f.key}`) }" />
      </NFormItem>
    </div>
  </CCard>
  <CCard v-if="budget" :title="t('common.result')" data-testid="result">
    <NAlert :type="budget.overload ? 'warning' : budget.headroom >= 0 ? 'success' : 'error'" class="mb-3" data-testid="verdict">
      {{ budget.overload ? t('tools.optical-power.overload', { db: budget.attenuatorDb }) : budget.headroom >= 0 ? t('tools.optical-power.ok', { db: budget.headroom }) : t('tools.optical-power.fail', { db: -budget.headroom }) }}
    </NAlert>
    <CKeyValueList :rows="budgetRows" />
  </CCard>
</template>
