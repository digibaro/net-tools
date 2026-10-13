<script setup lang="ts">
import type { Whois } from './whois-lookup.service'
import { NButton, NTabPane, NTabs } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { finalResponse, isRegistryQuery } from './whois-lookup.service'

const { t } = useI18n()
const query = useQueryParam({ name: 'query', defaultValue: '' })
const validation = useValidation({ source: query, rules: [{ validator: isRegistryQuery, message: t('registry.invalidQuery') }] })
const { data, error, loading, run } = useApi<Whois>('/v1/whois')
const submit = () => query.value && validation.value.isValid && run({ query: query.value.trim() })
onMounted(() => {
  if (query.value)
    void submit()
})
const rows = computed(() => {
  const p = data.value?.parsed
  if (!p)
    return []
  const k = (key: string) => t(`registry.fields.${key}`)
  return [
    { label: k('registrar'), value: p.registrar },
    { label: k('org'), value: p.org },
    { label: k('country'), value: p.country },
    { label: k('range'), value: p.netRange, mono: true },
    { label: k('created'), value: p.created, mono: true },
    { label: k('updated'), value: p.updated, mono: true },
    { label: k('expires'), value: p.expires, mono: true },
    { label: k('status'), value: p.status.join(', '), mono: true },
    { label: k('nameservers'), value: p.nameservers.join(', '), mono: true },
  ].filter(r => r.value)
})
const final = computed(() => (data.value ? finalResponse(data.value) : undefined))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="query" :label="t('registry.query')" placeholder="example.com, 8.8.8.8, AS13335" :validation="validation" monospace test-id="query" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!query || !validation.isValid" data-testid="run">
        {{ t('tools.whois-lookup.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard v-if="rows.length" :title="t('tools.whois-lookup.summary')" data-testid="summary">
      <CKeyValueList :rows="rows" />
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.whois-lookup.bestEffort') }}
      </p>
    </CCard>
    <CCard :title="t('tools.whois-lookup.raw')" data-testid="result">
      <p class="mt-0 text-xs nt-muted">
        {{ t('tools.whois-lookup.chain') }}: {{ data.chain.map(r => r.server).join(' → ') }}
      </p>
      <NTabs type="line" :default-value="final?.server" size="small">
        <NTabPane v-for="r in data.chain" :key="r.server" :name="r.server" :tab="r.server">
          <p v-if="r.error" class="m-0 text-sm">
            {{ r.error }}
          </p>
          <pre v-else class="m-0 max-h-120 overflow-auto whitespace-pre-wrap break-all text-xs font-mono" tabindex="0">{{ r.text }}</pre>
          <p v-if="r.truncated" class="text-xs nt-muted">
            {{ t('tools.whois-lookup.truncated') }}
          </p>
        </NTabPane>
      </NTabs>
      <div class="mt-3">
        <ResultActions :text="() => final?.text ?? ''" :json="data" filename="whois.json" />
      </div>
    </CCard>
  </template>
</template>
