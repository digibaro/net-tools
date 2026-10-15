<script setup lang="ts">
import type { Period } from './sla-calculator.service'
import { NFormItem, NInputNumber, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { fixed } from '@/utils/format'
import { availability, downtime, human, nines, parallel, parsePercent, PERIODS, series } from './sla-calculator.service'

const { t } = useI18n()
const sla = useQueryParam({ name: 'sla', defaultValue: '99.9' })
const validation = useValidation({ source: sla, rules: [{ validator: (s: string) => parsePercent(s) !== undefined, message: t('tools.sla-calculator.invalid') }] })
const pct = computed(() => parsePercent(sla.value))
const periods = Object.keys(PERIODS) as Period[]
const p = (k: string) => t(`tools.sla-calculator.periods.${k}`)
const rows = computed(() => (pct.value === undefined ? [] : periods.map(k => ({ period: p(k), downtime: human(downtime(pct.value!, k)) }))))
const minutes = useQueryParam({ name: 'down', defaultValue: 30 })
const per = useQueryParam({ name: 'per', defaultValue: 'month' })
const achieved = computed(() => availability(minutes.value * 60, per.value as Period))
const components = useQueryParam({ name: 'parts', defaultValue: '99.95, 99.9, 99.99' })
const mode = useQueryParam({ name: 'mode', defaultValue: 'series' })
const parts = computed(() => components.value.split(/[\s;]+|,\s+/).map(parsePercent).filter((x): x is number => x !== undefined))
const combined = computed(() => (parts.value.length ? (mode.value === 'parallel' ? parallel(parts.value) : series(parts.value)) : undefined))
const show = (v: number) => `${fixed(v, v > 99.99 ? 5 : 3)} % (${fixed(nines(v), 2)} ${t('tools.sla-calculator.ninesUnit')}) · ${human(downtime(v, 'year'))} / ${p('year')}`
</script>

<template>
  <CCard :title="t('tools.sla-calculator.downtimeTitle')">
    <CInputText v-model:value="sla" :label="t('tools.sla-calculator.sla')" placeholder="99.95" :validation="validation" monospace test-id="sla" />
    <ResultTable v-if="rows.length" :columns="[{ key: 'period', title: t('tools.sla-calculator.period'), width: 160 }, { key: 'downtime', title: t('tools.sla-calculator.allowed'), width: 200 }]" :rows="rows" row-key="period" filename="sla.json" data-testid="result" />
  </CCard>
  <CCard :title="t('tools.sla-calculator.achievedTitle')">
    <div class="flex flex-wrap items-end gap-x-4">
      <NFormItem :label="t('tools.sla-calculator.downMinutes')" :show-feedback="false" class="mb-3 w-40">
        <NInputNumber v-model:value="minutes" :min="0" :show-button="false" :input-props="{ 'aria-label': t('tools.sla-calculator.downMinutes') }" />
      </NFormItem>
      <NRadioGroup v-model:value="per" name="per" class="mb-3">
        <NRadioButton v-for="k in periods" :key="k" :value="k" :label="p(k)" />
      </NRadioGroup>
    </div>
    <p class="m-0 font-mono" data-testid="achieved">
      {{ show(achieved) }}
    </p>
  </CCard>
  <CCard :title="t('tools.sla-calculator.compositeTitle')">
    <CInputText v-model:value="components" :label="t('tools.sla-calculator.components')" placeholder="99.95, 99.9" monospace test-id="parts" />
    <NRadioGroup v-model:value="mode" name="mode" class="mb-3">
      <NRadioButton value="series" :label="t('tools.sla-calculator.series')" />
      <NRadioButton value="parallel" :label="t('tools.sla-calculator.parallel')" />
    </NRadioGroup>
    <CKeyValueList v-if="combined !== undefined" :rows="[{ label: t('tools.sla-calculator.combined'), value: show(combined), noCopy: true }]" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.sla-calculator.compositeHelp') }}
    </p>
  </CCard>
</template>
