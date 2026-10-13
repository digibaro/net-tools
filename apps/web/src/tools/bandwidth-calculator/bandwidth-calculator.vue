<script setup lang="ts">
import type { OverheadPreset, TimeUnit } from './bandwidth-calculator.service'
import { NFormItem, NInputNumber } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { formatDuration, humanize, RATE_UNITS, SIZE_UNITS, unitFactor } from '@/utils/units'
import { efficiency, OVERHEAD_PRESETS, requiredRate, TIME_UNITS, transferTime } from './bandwidth-calculator.service'

const { t } = useI18n()
const size = useQueryParam({ name: 'size', defaultValue: 4.7 })
const sizeUnit = useQueryParam({ name: 'su', defaultValue: 'GB' })
const rate = useQueryParam({ name: 'rate', defaultValue: 100 })
const rateUnit = useQueryParam({ name: 'ru', defaultValue: 'Mbps' })
const overhead = useQueryParam({ name: 'overhead', defaultValue: 'tcp-ipv4' })
const custom = useQueryParam({ name: 'custom', defaultValue: 5 })
const time = useQueryParam({ name: 'time', defaultValue: 10 })
const timeUnit = useQueryParam({ name: 'tu', defaultValue: 'min' })

const presets = computed(() => [...Object.keys(OVERHEAD_PRESETS), 'custom'].map(p => ({ label: t(`tools.bandwidth-calculator.presets.${p}`), value: p })))
const sizeOptions = SIZE_UNITS.map(u => ({ label: u.label, value: u.id }))
const rateOptions = RATE_UNITS.map(u => ({ label: u.label, value: u.id }))
const timeOptions = Object.keys(TIME_UNITS).map(u => ({ label: u, value: u }))

const eff = computed(() => efficiency(overhead.value as OverheadPreset, custom.value))
const sizeBits = computed(() => size.value * (unitFactor(SIZE_UNITS, sizeUnit.value) ?? Number.NaN))
const rateBps = computed(() => rate.value * (unitFactor(RATE_UNITS, rateUnit.value) ?? Number.NaN))
const seconds = computed(() => time.value * (TIME_UNITS[timeUnit.value as TimeUnit] ?? Number.NaN))

const duration = computed(() => transferTime(sizeBits.value, rateBps.value, eff.value))
const needed = computed(() => requiredRate(sizeBits.value, seconds.value, eff.value))

const k = (key: string) => t(`tools.bandwidth-calculator.fields.${key}`)
const timeRows = computed(() => duration.value === undefined
  ? []
  : [
      { label: k('time'), value: formatDuration(duration.value) },
      { label: k('goodput'), value: humanize(rateBps.value * eff.value, 'rate-bits') },
      { label: k('efficiency'), value: `${(eff.value * 100).toFixed(2)} %`, noCopy: true },
    ])
const rateRows = computed(() => needed.value === undefined
  ? []
  : [
      { label: k('requiredRate'), value: humanize(needed.value, 'rate-bits') },
      { label: k('requiredRateBytes'), value: humanize(needed.value, 'rate-bytes') },
    ])
</script>

<template>
  <CCard>
    <div class="grid gap-x-4 sm:grid-cols-2">
      <div class="flex gap-2">
        <NFormItem :label="t('tools.bandwidth-calculator.size')" :show-feedback="false" class="mb-3 flex-1">
          <NInputNumber v-model:value="size" :min="0" :show-button="false" :input-props="{ 'aria-label': t('tools.bandwidth-calculator.size') }" data-testid="size" />
        </NFormItem>
        <div class="w-32">
          <CSelect v-model:value="sizeUnit" :label="t('tools.bandwidth-calculator.unit')" :options="sizeOptions" />
        </div>
      </div>
      <div>
        <CSelect v-model:value="overhead" :label="t('tools.bandwidth-calculator.overhead')" :options="presets" />
        <NFormItem v-if="overhead === 'custom'" :label="t('tools.bandwidth-calculator.customPercent')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="custom" :min="0" :max="99" :show-button="false" :input-props="{ 'aria-label': t('tools.bandwidth-calculator.customPercent') }" />
        </NFormItem>
      </div>
    </div>
  </CCard>
  <div class="grid gap-x-4 md:grid-cols-2">
    <CCard :title="t('tools.bandwidth-calculator.timeTitle')" data-testid="time-result">
      <div class="flex gap-2">
        <NFormItem :label="t('tools.bandwidth-calculator.rate')" :show-feedback="false" class="mb-3 flex-1">
          <NInputNumber v-model:value="rate" :min="0" :show-button="false" :input-props="{ 'aria-label': t('tools.bandwidth-calculator.rate') }" data-testid="rate" />
        </NFormItem>
        <div class="w-32">
          <CSelect v-model:value="rateUnit" :label="t('tools.bandwidth-calculator.unit')" :options="rateOptions" />
        </div>
      </div>
      <CKeyValueList :rows="timeRows" />
    </CCard>
    <CCard :title="t('tools.bandwidth-calculator.rateTitle')" data-testid="rate-result">
      <div class="flex gap-2">
        <NFormItem :label="t('tools.bandwidth-calculator.time')" :show-feedback="false" class="mb-3 flex-1">
          <NInputNumber v-model:value="time" :min="0" :show-button="false" :input-props="{ 'aria-label': t('tools.bandwidth-calculator.time') }" />
        </NFormItem>
        <div class="w-32">
          <CSelect v-model:value="timeUnit" :label="t('tools.bandwidth-calculator.unit')" :options="timeOptions" />
        </div>
      </div>
      <CKeyValueList :rows="rateRows" />
    </CCard>
  </div>
</template>
