<script setup lang="ts">
import type { Validation } from '@/composables/useValidation'
import { NFormItem, NProgress, NSwitch } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { formatCount } from '@/utils/ip'
import { MAX_SUBNETS, plan } from './vlsm-planner.service'

const { t } = useI18n()
const network = useQueryParam({ name: 'net', defaultValue: '192.168.1.0/24' })
const reqs = useQueryParam({ name: 'hosts', defaultValue: 'Office 100\nWiFi 50\nServers 20\nLink 2' })
const p2p = useQueryParam({ name: 'p2p', defaultValue: true })
const result = computed(() => plan(network.value, reqs.value, p2p.value))
const ok = computed(() => (typeof result.value === 'string' ? undefined : result.value))
const err = computed(() => (typeof result.value === 'string' ? result.value : undefined))
function fieldValidation(code: 'network' | 'requirements'): Validation {
  return err.value === code
    ? { isValid: false, message: t(`tools.vlsm-planner.errors.${code}`), status: 'error' }
    : { isValid: true, message: '', status: undefined }
}
const networkValidation = computed(() => fieldValidation('network'))
const reqValidation = computed(() => fieldValidation('requirements'))
const rows = computed(() => (ok.value?.allocations ?? []).map(a => ({ ...a, efficiency: `${Math.round(a.efficiency * 100)} %` })))
const k = (key: string) => t(`tools.vlsm-planner.columns.${key}`)
const columns = computed(() => [
  { key: 'name', title: k('name'), width: 140 },
  { key: 'hosts', title: k('hosts'), width: 90 },
  { key: 'cidr', title: k('cidr'), mono: true, width: 200 },
  { key: 'first', title: k('first'), mono: true, width: 170 },
  { key: 'last', title: k('last'), mono: true, width: 170 },
  { key: 'usable', title: k('usable'), width: 100 },
  { key: 'efficiency', title: k('efficiency'), width: 100 },
])
const usedPct = computed(() => (ok.value ? Number(ok.value.used * 10000n / ok.value.total) / 100 : 0))
</script>

<template>
  <CCard>
    <CInputText v-model:value="network" :label="t('tools.vlsm-planner.network')" placeholder="10.0.0.0/16" :validation="networkValidation" monospace test-id="network" />
    <CInputText v-model:value="reqs" :label="t('tools.vlsm-planner.requirements')" :placeholder="t('tools.vlsm-planner.requirementsPlaceholder')" :validation="reqValidation" multiline :rows="6" monospace test-id="requirements" />
    <NFormItem :label="t('tools.vlsm-planner.p2p')" :show-feedback="false">
      <NSwitch v-model:value="p2p" :aria-label="t('tools.vlsm-planner.p2p')" />
    </NFormItem>
  </CCard>
  <p v-if="err === 'no-fit' || err === 'too-many'" class="nt-muted" data-testid="no-result">
    {{ t(`tools.vlsm-planner.errors.${err}`, { max: MAX_SUBNETS }) }}
  </p>
  <CCard v-if="ok" :title="t('common.result')" data-testid="result">
    <div class="mb-3 max-w-md">
      <div class="mb-1 text-sm">
        {{ t('tools.vlsm-planner.used', { used: formatCount(ok.used), total: formatCount(ok.total), pct: usedPct }) }}
      </div>
      <NProgress type="line" :percentage="usedPct" :show-indicator="false" :aria-label="t('tools.vlsm-planner.used', { used: formatCount(ok.used), total: formatCount(ok.total), pct: usedPct })" />
    </div>
    <ResultTable :columns="columns" :rows="rows" row-key="cidr" filename="vlsm.json" />
    <template v-if="ok.free.length">
      <h3 class="mb-1 mt-4 text-sm font-semibold">
        {{ t('tools.vlsm-planner.free') }}
      </h3>
      <p class="m-0 break-all text-sm font-mono" data-testid="free">
        {{ ok.free.join(', ') }}
      </p>
    </template>
  </CCard>
</template>
