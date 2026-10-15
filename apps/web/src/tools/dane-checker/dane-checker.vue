<script setup lang="ts">
import type { DaneHost, DaneResult } from './dane-checker.service'
import { NAlert, NButton, NFormItem, NRadioButton, NRadioGroup, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { isResolver } from '../nat64-dns64-check/nat64-dns64-check.service'
import { describeRecord, overall } from './dane-checker.service'

const { t } = useI18n()
const host = useQueryParam({ name: 'host', defaultValue: '' })
const service = useQueryParam({ name: 'service', defaultValue: 'smtp' })
const resolver = useQueryParam({ name: 'resolver', defaultValue: '' })
const validation = useValidation({ source: host, rules: [{ validator: (s: string) => !s || isValidQueryName(s), message: t('tools.dns-lookup.invalidName') }] })
const resolverValidation = useValidation({ source: resolver, rules: [{ validator: isResolver, message: t('tools.nat64-dns64-check.invalidResolver') }] })
const { data, error, loading, run } = useApi<DaneResult>('/v1/tls/dane')
const submit = () => host.value && validation.value.isValid && resolverValidation.value.isValid && run({ host: host.value.trim(), service: service.value, resolver: resolver.value.trim() || 'system' })
onMounted(() => {
  if (host.value)
    void submit()
})
const verdictType = { 'valid': 'success', 'mismatch': 'error', 'insecure': 'warning', 'no-tlsa': 'info', 'unreachable': 'warning' } as const
const o = computed(() => (data.value ? overall(data.value) : undefined))
const k = (key: string) => t(`tools.dane-checker.fields.${key}`)
function rows(h: DaneHost) {
  return [
    { label: k('tlsaName'), value: h.tlsaName, mono: true },
    { label: k('dnssec'), value: h.dnssec ? t('tools.dane-checker.dnssecYes') : t('tools.dane-checker.dnssecNo'), noCopy: true },
    { label: k('address'), value: h.address || '—', mono: true },
    ...(h.tls ? [{ label: k('tls'), value: `${h.tls.version} · ${h.tls.cipher}`, noCopy: true }, { label: k('webpki'), value: h.tls.verified ? t('common.yes') : `${t('common.no')} (${h.tls.verifyError})`, noCopy: true }] : []),
  ]
}
const recordRows = (h: DaneHost) => h.records.map((r, i) => ({ id: String(i), params: describeRecord(r), data: r.data, match: r.match ? t('tools.dane-checker.matchCert', { n: r.matchedCert === 0 ? t('tools.dane-checker.leaf') : `#${r.matchedCert}` }) : t('tools.dane-checker.noMatch') }))
const c = (key: string) => t(`tools.dane-checker.columns.${key}`)
const asText = () => (data.value?.hosts ?? []).map(h => `${h.name}: ${h.verdict} (${h.records.length} TLSA, DNSSEC ${h.dnssec})`).join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="host" :label="service === 'smtp' ? t('tools.dane-checker.domain') : t('tools.dane-checker.host')" placeholder="example.com" :validation="validation" monospace test-id="host" />
        <CInputText v-model:value="resolver" :label="t('tools.nat64-dns64-check.resolver')" :placeholder="t('tools.dane-checker.resolverPlaceholder')" :validation="resolverValidation" monospace test-id="resolver" />
      </div>
      <NFormItem :label="t('tools.dane-checker.service')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="service" name="service">
          <NRadioButton value="smtp" :label="t('tools.dane-checker.smtp')" />
          <NRadioButton value="https" label="HTTPS (443)" />
        </NRadioGroup>
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!host || !validation.isValid" data-testid="run">
        {{ t('tools.dane-checker.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.dane-checker.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <NAlert v-if="o" :type="verdictType[o]" class="mb-4" data-testid="verdict">
    {{ t(`tools.dane-checker.verdict.${o}`, { input: data!.input }) }}
  </NAlert>
  <CCard v-for="h in data?.hosts ?? []" :key="h.name" :title="h.name" data-testid="result">
    <NTag :type="verdictType[h.verdict]" :bordered="false" class="mb-3">
      {{ t(`tools.dane-checker.hostVerdict.${h.verdict}`) }}
    </NTag>
    <NAlert v-if="h.error" type="warning" class="mb-3">
      {{ h.error }}
    </NAlert>
    <CKeyValueList :rows="rows(h)" />
    <ResultTable v-if="h.records.length" class="mt-3" :columns="[{ key: 'params', title: c('params'), width: 240 }, { key: 'data', title: c('data'), mono: true, width: 420 }, { key: 'match', title: c('match'), width: 160 }]" :rows="recordRows(h)" row-key="id" :actions="false" :label="h.tlsaName" />
  </CCard>
  <div v-if="data?.hosts.length" class="mb-4">
    <ResultActions :text="asText" :json="data" filename="dane.json" />
  </div>
</template>
