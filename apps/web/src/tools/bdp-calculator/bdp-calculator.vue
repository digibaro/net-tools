<script setup lang="ts">
import { NFormItem, NInputNumber } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { formatNumber, humanize, RATE_UNITS, unitFactor } from '@/utils/units'
import { bdp, mathisThroughput, MAX_UNSCALED_WINDOW, windowThroughput } from './bdp-calculator.service'

const { t } = useI18n()
const rate = useQueryParam({ name: 'rate', defaultValue: 1 })
const rateUnit = useQueryParam({ name: 'ru', defaultValue: 'Gbps' })
const rtt = useQueryParam({ name: 'rtt', defaultValue: 50 })
const windowKiB = useQueryParam({ name: 'window', defaultValue: 64 })
const mss = useQueryParam({ name: 'mss', defaultValue: 1460 })
const loss = useQueryParam({ name: 'loss', defaultValue: 0.1 })
const rateOptions = RATE_UNITS.map(u => ({ label: u.label, value: u.id }))

const k = (key: string) => t(`tools.bdp-calculator.fields.${key}`)
const bdpResult = computed(() => bdp(rate.value * (unitFactor(RATE_UNITS, rateUnit.value) ?? Number.NaN), rtt.value))
const bdpRows = computed(() => {
  const r = bdpResult.value
  if (!r)
    return []
  return [
    { label: k('bdp'), value: `${humanize(r.bdpBits, 'size-bytes')} (${formatNumber(r.bdpBytes, 10)} bytes)` },
    { label: k('window'), value: `${formatNumber(Math.ceil(r.bdpBytes), 12)} bytes` },
    { label: k('scaling'), value: r.bdpBytes > MAX_UNSCALED_WINDOW ? t('tools.bdp-calculator.scalingNeeded', { shift: r.windowScale ?? '> 14' }) : t('tools.bdp-calculator.scalingNotNeeded'), noCopy: true },
  ]
})
const winRows = computed(() => {
  const w = windowThroughput(windowKiB.value * 1024, rtt.value)
  const m = mathisThroughput(mss.value, rtt.value, loss.value)
  return [
    ...(w === undefined ? [] : [{ label: k('maxThroughput'), value: humanize(w, 'rate-bits') }]),
    ...(m === undefined ? [] : [{ label: k('mathis'), value: humanize(m, 'rate-bits') }]),
  ]
})
</script>

<template>
  <CCard>
    <div class="grid gap-x-4 sm:grid-cols-[1fr_9rem_1fr]">
      <NFormItem :label="t('tools.bdp-calculator.rate')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="rate" :min="0" :show-button="false" :input-props="{ 'aria-label': t('tools.bdp-calculator.rate') }" />
      </NFormItem>
      <CSelect v-model:value="rateUnit" :label="t('tools.bdp-calculator.unit')" :options="rateOptions" />
      <NFormItem :label="t('tools.bdp-calculator.rtt')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="rtt" :min="0" :show-button="false" :input-props="{ 'aria-label': t('tools.bdp-calculator.rtt') }" />
      </NFormItem>
    </div>
  </CCard>
  <CCard :title="t('tools.bdp-calculator.bdpTitle')" data-testid="bdp-result">
    <CKeyValueList :rows="bdpRows" />
  </CCard>
  <CCard :title="t('tools.bdp-calculator.windowTitle')" data-testid="window-result">
    <div class="grid gap-x-4 sm:grid-cols-3">
      <NFormItem :label="t('tools.bdp-calculator.windowKiB')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="windowKiB" :min="0" :show-button="false" :input-props="{ 'aria-label': t('tools.bdp-calculator.windowKiB') }" />
      </NFormItem>
      <NFormItem :label="t('tools.bdp-calculator.mss')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="mss" :min="1" :show-button="false" :input-props="{ 'aria-label': t('tools.bdp-calculator.mss') }" />
      </NFormItem>
      <NFormItem :label="t('tools.bdp-calculator.loss')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="loss" :min="0" :max="99" :show-button="false" :input-props="{ 'aria-label': t('tools.bdp-calculator.loss') }" />
      </NFormItem>
    </div>
    <CKeyValueList :rows="winRows" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.bdp-calculator.mathisNote') }}
    </p>
  </CCard>
</template>
