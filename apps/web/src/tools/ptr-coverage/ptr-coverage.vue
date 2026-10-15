<script setup lang="ts">
import type { PtrScanResult } from './ptr-coverage.service'
import { NButton, NSwitch } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isResolver } from '../nat64-dns64-check/nat64-dns64-check.service'
import { isScanPrefix, status } from './ptr-coverage.service'

const { t } = useI18n()
const prefix = useQueryParam({ name: 'prefix', defaultValue: '' })
const resolver = useQueryParam({ name: 'resolver', defaultValue: '' })
const onlyNamed = useQueryParam({ name: 'named', defaultValue: true })
const validation = useValidation({ source: prefix, rules: [{ validator: (s: string) => !s || isScanPrefix(s), message: t('tools.ptr-coverage.invalid') }] })
const resolverValidation = useValidation({ source: resolver, rules: [{ validator: isResolver, message: t('tools.nat64-dns64-check.invalidResolver') }] })
const { data, error, loading, run } = useApi<PtrScanResult>('/v1/dns/ptr-scan')
const submit = () => prefix.value && validation.value.isValid && resolverValidation.value.isValid && run({ prefix: prefix.value.trim(), resolver: resolver.value.trim() || 'system' })
onMounted(() => {
  if (prefix.value)
    void submit()
})
const rows = computed(() => (data.value?.rows ?? [])
  .filter(r => !onlyNamed.value || r.names.length)
  .map(r => ({ ip: r.ip, names: r.names.join(', ') || '—', status: t(`tools.ptr-coverage.status.${status(r)}`), rcode: r.rcode || r.error })))
const k = (key: string) => t(`tools.ptr-coverage.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="prefix" :label="t('tools.ptr-coverage.prefix')" placeholder="192.0.2.0/24" :validation="validation" monospace test-id="prefix" />
        <CInputText v-model:value="resolver" :label="t('tools.nat64-dns64-check.resolver')" :placeholder="t('tools.nat64-dns64-check.resolverPlaceholder')" :validation="resolverValidation" monospace test-id="resolver" />
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!prefix || !validation.isValid" data-testid="run">
        {{ t('tools.ptr-coverage.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('tools.ptr-coverage.summary', { prefix: data.prefix, ptr: data.withPtr, total: data.total, confirmed: data.confirmed })" data-testid="result">
    <label class="mb-3 flex items-center gap-2 text-sm">
      <NSwitch v-model:value="onlyNamed" size="small" />
      {{ t('tools.ptr-coverage.onlyNamed') }}
    </label>
    <ResultTable :columns="[{ key: 'ip', title: k('ip'), mono: true, width: 200 }, { key: 'names', title: k('names'), mono: true, width: 300 }, { key: 'status', title: k('status'), width: 200 }, { key: 'rcode', title: k('rcode'), width: 110 }]" :rows="rows" row-key="ip" filename="ptr-coverage.json" />
  </CCard>
</template>
