<script setup lang="ts">
import type { ChainResult } from './cert-chain-validator.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NFormItem, NInput } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiPost } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { countCerts, verdict } from './cert-chain-validator.service'

const { t } = useI18n()
const pem = ref('')
const host = useQueryParam({ name: 'host', defaultValue: '' })
const hostValidation = useValidation({ source: host, rules: [{ validator: (s: string) => !s || isValidQueryName(s), message: t('tools.dns-lookup.invalidName') }] })
const result = shallowRef<ChainResult>()
const error = ref<ApiErrorType>()
const loading = ref(false)
const n = computed(() => countCerts(pem.value))
async function submit() {
  if (!n.value || !hostValidation.value.isValid)
    return
  loading.value = true
  error.value = undefined
  result.value = undefined
  try {
    result.value = await apiPost<ChainResult>('/v1/tls/chain-validate', { pem: pem.value, host: host.value.trim() })
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    loading.value = false
  }
}
const v = computed(() => (result.value ? verdict(result.value) : undefined))
const alertType = { ok: 'success', unordered: 'info', incomplete: 'warning', untrusted: 'error' } as const
const rows = computed(() => (result.value?.path ?? []).map((i, k) => {
  const c = result.value!.certs[i]!
  return {
    id: String(i),
    n: k + 1,
    role: t(`tools.cert-chain-validator.roles.${c.role}`),
    subject: c.subject,
    expires: `${c.notAfter.slice(0, 10)}${c.expired ? ` (${t('tools.cert-chain-validator.expired')})` : ''}`,
    source: c.source === 'aia' ? t('tools.cert-chain-validator.fetched') : t('tools.cert-chain-validator.pasted', { n: c.inputIndex + 1 }),
    signature: c.signatureOk === null ? '—' : c.signatureOk ? '✓' : '✗',
  }
}))
const extra = computed(() => (result.value?.certs ?? []).filter(c => c.role === 'extra'))
const k = (key: string) => t(`tools.cert-chain-validator.columns.${key}`)
const asText = () => [v.value ? t(`tools.cert-chain-validator.verdict.${v.value}`, { error: result.value?.verifyError }) : '', ...rows.value.map(r => `${r.n}. ${r.role}: ${r.subject} (${r.expires})`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <NFormItem :label="t('tools.cert-chain-validator.pem')" :show-feedback="false" class="mb-1">
        <NInput v-model:value="pem" type="textarea" :rows="8" class="font-mono" placeholder="-----BEGIN CERTIFICATE-----" :input-props="{ 'aria-label': t('tools.cert-chain-validator.pem'), 'spellcheck': false }" data-testid="pem" />
      </NFormItem>
      <p class="mb-3 mt-0 text-xs nt-muted">
        {{ t('tools.cert-chain-validator.found', { n }) }}
      </p>
      <CInputText v-model:value="host" :label="t('tools.cert-chain-validator.host')" placeholder="www.example.com" :validation="hostValidation" monospace test-id="host" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!n || !hostValidation.isValid" data-testid="run">
        {{ t('tools.cert-chain-validator.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="result && v" :title="t('common.result')" data-testid="result">
    <NAlert :type="alertType[v]" class="mb-3" data-testid="verdict">
      {{ t(`tools.cert-chain-validator.verdict.${v}`, { error: result.verifyError }) }}
    </NAlert>
    <NAlert v-if="result.hostOk !== null" :type="result.hostOk ? 'success' : 'error'" class="mb-3">
      {{ t(result.hostOk ? 'tools.cert-chain-validator.hostOk' : 'tools.cert-chain-validator.hostBad', { host: result.host }) }}
    </NAlert>
    <ResultTable :columns="[{ key: 'n', title: '#', width: 40 }, { key: 'role', title: k('role'), width: 120 }, { key: 'subject', title: k('subject'), width: 320 }, { key: 'expires', title: k('expires'), width: 150 }, { key: 'source', title: k('source'), width: 150 }, { key: 'signature', title: k('signature'), width: 90 }]" :rows="rows" row-key="id" filename="chain.json" />
    <p v-if="extra.length" class="text-sm nt-muted">
      {{ t('tools.cert-chain-validator.extra', { names: extra.map(c => c.subject).join('; ') }) }}
    </p>
    <p v-for="note in result.notes.filter(x => x !== 'missing-intermediates')" :key="note" class="m-0 text-xs nt-muted">
      {{ note }}
    </p>
    <div class="mt-3">
      <ResultActions :text="asText" :json="result" filename="chain-validation.json" />
    </div>
  </CCard>
</template>
