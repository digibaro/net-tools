<script setup lang="ts">
import type { Cors } from './cors-tester.service'
import { NAlert, NButton, NFormItem, NSwitch, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import HttpHopView from '@/components/HttpHopView.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { isValidHttpUrl } from '@/utils/url'
import { METHODS } from '../http-request/http-request.service'
import { isValidHeaderList, isValidOrigin } from './cors-tester.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const origin = useQueryParam({ name: 'origin', defaultValue: 'https://app.example.com' })
const method = useQueryParam({ name: 'method', defaultValue: 'GET' })
const headerList = useQueryParam({ name: 'headers', defaultValue: '' })
const credentials = useQueryParam({ name: 'credentials', defaultValue: false })
const urlV = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const originV = useValidation({ source: origin, rules: [{ validator: isValidOrigin, message: t('tools.cors-tester.invalidOrigin') }] })
const headersV = useValidation({ source: headerList, rules: [{ validator: isValidHeaderList, message: t('tools.cors-tester.invalidHeaders') }] })
const canRun = computed(() => !!url.value && urlV.value.isValid && originV.value.isValid && headersV.value.isValid)
const { data, error, loading, run } = useApi<Cors>('/v1/http/cors')
const submit = () => canRun.value && run({ url: url.value, origin: origin.value, method: method.value, headers: headerList.value || undefined, credentials: credentials.value || undefined })
onMounted(() => {
  if (url.value)
    void submit()
})
const methodOptions = METHODS.map(m => ({ label: m, value: m }))
const asText = () => [t(data.value?.allowed ? 'tools.cors-tester.allowed' : 'tools.cors-tester.blocked'), ...(data.value?.findings ?? []).map(f => `[${f.ok ? 'ok' : 'fail'}] ${t(`tools.cors-tester.findings.${f.code}`)}: ${f.value || '—'}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://api.example.com/resource" :validation="urlV" monospace test-id="url" />
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="origin" :label="t('tools.cors-tester.origin')" :validation="originV" monospace test-id="origin" />
        <CSelect v-model:value="method" :label="t('tools.http-request.method')" :options="methodOptions" />
      </div>
      <CInputText v-model:value="headerList" :label="t('tools.cors-tester.headers')" placeholder="Authorization, Content-Type" :validation="headersV" monospace />
      <NFormItem :label="t('tools.cors-tester.credentials')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="credentials" :aria-label="t('tools.cors-tester.credentials')" />
      </NFormItem>
      <p class="mt-0 text-xs nt-muted">
        {{ t('tools.cors-tester.note') }}
      </p>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!canRun" data-testid="run">
        {{ t('tools.cors-tester.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.allowed ? 'success' : 'error'" class="mb-4" data-testid="verdict">
      {{ t(data.allowed ? 'tools.cors-tester.allowed' : 'tools.cors-tester.blocked') }}
      {{ data.needsPreflight ? t('tools.cors-tester.preflightSent') : t('tools.cors-tester.simple') }}
    </NAlert>
    <CCard :title="t('tools.cors-tester.evaluation')" data-testid="result">
      <ul class="m-0 list-none p-0">
        <li v-for="f in data.findings" :key="f.code" class="flex flex-wrap items-center gap-2 py-1">
          <NTag :type="f.ok ? 'success' : f.code === 'vary-origin' || f.code === 'max-age' ? 'warning' : 'error'" size="small" :bordered="false">
            {{ f.ok ? 'OK' : '✕' }}
          </NTag>
          <span class="text-sm">{{ t(`tools.cors-tester.findings.${f.code}`) }}</span>
          <span class="break-all text-xs nt-muted font-mono">{{ f.value || '—' }}</span>
        </li>
      </ul>
      <div class="mt-3">
        <ResultActions :text="asText" :json="data" filename="cors.json" />
      </div>
    </CCard>
    <CCard v-if="data.preflight" :title="t('tools.cors-tester.preflight')">
      <HttpHopView :hop="data.preflight" />
    </CCard>
    <CCard v-if="data.actual" :title="t('tools.cors-tester.actual')">
      <HttpHopView :hop="data.actual" />
    </CCard>
  </template>
</template>
