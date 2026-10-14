<script setup lang="ts">
import type { TlsExpiry } from './tls-expiry.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NButton } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiPost } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { parseTargets, sortRows } from './tls-expiry.service'

const { t } = useI18n()
const caps = useCapabilitiesStore()
const input = useQueryParam({ name: 'hosts', defaultValue: '' })
const targets = computed(() => parseTargets(input.value))
const max = computed(() => caps.capabilities?.limits.bulkMaxTargets ?? 50)
const validation = computed(() => targets.value.length > max.value ? { isValid: false, status: 'error' as const, message: t('tools.tls-expiry.tooMany', { max: max.value }) } : undefined)
const loading = ref(false)
const data = shallowRef<TlsExpiry>()
const error = shallowRef<ApiErrorType>()
async function run() {
  if (!targets.value.length || validation.value)
    return
  loading.value = true
  error.value = undefined
  try {
    data.value = await apiPost<TlsExpiry>('/v1/tls/expiry', { targets: targets.value })
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('BAD_RESPONSE', String(e))
  }
  finally {
    loading.value = false
  }
}
const rows = computed(() => sortRows(data.value?.rows ?? []).map(r => ({
  target: r.target,
  days: r.daysLeft === null ? '—' : r.daysLeft,
  expires: r.notAfter ? r.notAfter.slice(0, 10) : '',
  status: r.error ? `${r.errorCode}: ${r.error}` : r.verified ? t('tools.tls-expiry.valid') : r.verifyError,
  issuer: r.issuer.replace(/^.*CN=/, ''),
})))
const k = (key: string) => t(`tools.tls-expiry.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="input" :label="t('tools.tls-expiry.input')" placeholder="example.com&#10;mail.example.com:465" :validation="validation" multiline :rows="6" monospace test-id="hosts" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!targets.length || !!validation" data-testid="run">
        {{ t('tools.tls-expiry.run', { count: targets.length }) }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <ResultTable :columns="[{ key: 'target', title: k('target'), mono: true, width: 220 }, { key: 'days', title: k('days'), width: 90, sortable: true }, { key: 'expires', title: k('expires'), width: 110 }, { key: 'issuer', title: k('issuer'), width: 200 }, { key: 'status', title: k('status'), width: 280 }]" :rows="rows" filename="tls-expiry.json" />
  </CCard>
</template>
