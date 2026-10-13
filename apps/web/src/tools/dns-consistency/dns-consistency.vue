<script setup lang="ts">
import type { DnsConsistency } from './dns-consistency.service'
import { NButton, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { checks } from './dns-consistency.service'

const { t } = useI18n()
const zone = useQueryParam({ name: 'zone', defaultValue: '' })
const validation = useValidation({ source: zone, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<DnsConsistency>('/v1/dns/consistency')
const submit = () => zone.value && validation.value.isValid && run({ zone: zone.value })
onMounted(() => {
  if (zone.value)
    void submit()
})

const rows = computed(() => (data.value?.servers ?? []).map(s => ({
  name: s.name,
  address: s.address || '—',
  serial: s.serial ?? '—',
  aa: s.serial === null ? '—' : t(s.authoritative ? 'common.yes' : 'common.no'),
  ns: s.nsSet.join(' ') || '—',
  rtt: s.rttMs ? `${s.rttMs} ms` : '—',
  error: s.error,
})))
const k = (key: string) => t(`tools.dns-consistency.columns.${key}`)
const columns = computed(() => [
  { key: 'name', title: k('name'), mono: true, width: 200 },
  { key: 'address', title: k('address'), mono: true, width: 180 },
  { key: 'serial', title: k('serial'), mono: true, width: 120, sortable: true },
  { key: 'aa', title: k('aa'), width: 70 },
  { key: 'rtt', title: k('rtt'), width: 90 },
  { key: 'ns', title: k('ns'), mono: true, width: 280 },
  { key: 'error', title: k('error'), width: 220 },
])
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="zone" :label="t('tools.dns-consistency.zone')" placeholder="example.com" :validation="validation" monospace test-id="zone" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!zone || !validation.isValid" data-testid="run">
        {{ t('tools.dns-consistency.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <div class="mb-3 flex flex-wrap gap-2">
      <NTag v-for="c in checks(data)" :key="c.id" :type="c.ok ? 'success' : 'error'" :bordered="false" :data-testid="`check-${c.id}`">
        {{ t(`tools.dns-consistency.checks.${c.id}.${c.ok ? 'ok' : 'fail'}`) }}
      </NTag>
    </div>
    <ResultTable :columns="columns" :rows="rows" filename="dns-consistency.json" />
  </CCard>
</template>
