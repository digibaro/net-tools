<script setup lang="ts">
import type { Validation } from '@/composables/useValidation'
import { NAlert, NProgress } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { formatCount } from '@/utils/ip'
import { buildPlan, MAX_ROWS } from './ipam-sheet.service'

const { t } = useI18n()
const block = useQueryParam({ name: 'block', defaultValue: '10.10.0.0/22' })
const planText = useQueryParam({ name: 'plan', defaultValue: 'Management, 99, /28, 10.10.0.0/28\nServers, 10, /24\nUsers, 20, 200 hosts, , floor 2\nGuest Wi-Fi, 30, /25\nUplink, , /31' })
const result = computed(() => buildPlan(block.value, planText.value))
const ok = computed(() => (typeof result.value === 'string' ? undefined : result.value))
const blockValidation = computed<Validation>(() => result.value === 'block'
  ? { isValid: false, message: t('tools.ipam-sheet.invalidBlock'), status: 'error' }
  : { isValid: true, message: '', status: undefined })
const k = (key: string) => t(`tools.ipam-sheet.columns.${key}`)
const rows = computed(() => (ok.value?.rows ?? []).map(r => ({
  ...r,
  id: String(r.line),
  cidr: r.cidr ?? '—',
  status: r.error ? t(`tools.ipam-sheet.errors.${r.error}`) : r.fixed ? t('tools.ipam-sheet.fixed') : t('tools.ipam-sheet.auto'),
})))
const columns = computed(() => [
  { key: 'name', title: k('name'), width: 150 },
  { key: 'vlan', title: 'VLAN', width: 70 },
  { key: 'cidr', title: k('cidr'), mono: true, width: 190 },
  { key: 'gateway', title: k('gateway'), mono: true, width: 170 },
  { key: 'range', title: k('range'), mono: true, width: 300 },
  ...(ok.value?.version === 4 ? [{ key: 'mask', title: k('mask'), mono: true, width: 140 }] : []),
  { key: 'usable', title: k('usable'), width: 110 },
  { key: 'status', title: k('status'), width: 160 },
  { key: 'note', title: k('note'), width: 160 },
])
const errors = computed(() => (ok.value?.rows ?? []).filter(r => r.error).length)
const usedPct = computed(() => (ok.value ? Number(ok.value.used * 10000n / ok.value.total) / 100 : 0))
const usedText = computed(() => ok.value ? t('tools.ipam-sheet.used', { used: formatCount(ok.value.used), total: formatCount(ok.value.total), pct: usedPct.value }) : '')
</script>

<template>
  <CCard>
    <CInputText v-model:value="block" :label="t('tools.ipam-sheet.block')" placeholder="10.10.0.0/22 or 2001:db8:100::/56" :validation="blockValidation" monospace test-id="block" />
    <CInputText v-model:value="planText" :label="t('tools.ipam-sheet.plan')" :placeholder="t('tools.ipam-sheet.planPlaceholder')" multiline :rows="8" monospace test-id="plan" />
    <p class="nt-muted m-0 text-xs">
      {{ t('tools.ipam-sheet.help') }}
    </p>
  </CCard>
  <p v-if="result === 'too-many'" class="nt-muted" data-testid="no-result">
    {{ t('tools.ipam-sheet.tooMany', { max: MAX_ROWS }) }}
  </p>
  <CCard v-if="ok" :title="t('tools.ipam-sheet.result', { block: ok.block })" data-testid="result">
    <NAlert v-if="errors" type="warning" class="mb-3">
      {{ t('tools.ipam-sheet.someErrors', { n: errors }) }}
    </NAlert>
    <div class="mb-3 max-w-md">
      <div class="mb-1 text-sm">
        {{ usedText }}
      </div>
      <NProgress type="line" :percentage="usedPct" :show-indicator="false" :aria-label="usedText" />
    </div>
    <ResultTable :columns="columns" :rows="rows" row-key="id" filename="ipam-plan.json" />
    <h3 class="mb-1 mt-4 text-sm font-semibold">
      {{ t('tools.ipam-sheet.free', { n: ok.free.length }) }}
    </h3>
    <ResultTable v-if="ok.free.length" :columns="[{ key: 'cidr', title: k('cidr'), mono: true, width: 240 }]" :rows="ok.free.map(cidr => ({ cidr }))" row-key="cidr" filename="ipam-free.json" />
  </CCard>
</template>
