<script setup lang="ts">
import type { HttpHops } from './http-request.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NButton, NFormItem, NSwitch } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiPost } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import HttpHopView from '@/components/HttpHopView.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { isValidHttpUrl } from '@/utils/url'
import { BODY_METHODS, METHODS, parseHeaderLines } from './http-request.service'

// Headers and body may contain secrets, so only URL, method and the follow flag go into the page URL.
const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const method = useQueryParam({ name: 'method', defaultValue: 'GET' })
const follow = useQueryParam({ name: 'follow', defaultValue: false })
const headerText = ref('')
const body = ref('')
const urlValidation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const parsed = computed(() => parseHeaderLines(headerText.value))
const headerValidation = computed(() => ('error' in parsed.value ? { isValid: false, status: 'error' as const, message: t('tools.http-request.invalidHeader', { line: parsed.value.error }) } : undefined))
const methodOptions = METHODS.map(m => ({ label: m, value: m }))

const loading = ref(false)
const data = shallowRef<HttpHops>()
const error = shallowRef<ApiErrorType>()
const canRun = computed(() => !!url.value && urlValidation.value.isValid && !('error' in parsed.value))
async function run() {
  if (!canRun.value || 'error' in parsed.value)
    return
  loading.value = true
  error.value = undefined
  try {
    data.value = await apiPost<HttpHops>('/v1/http/request', {
      url: url.value,
      method: method.value,
      headers: parsed.value.headers,
      body: BODY_METHODS.includes(method.value) ? body.value : '',
      followRedirects: follow.value,
    })
  }
  catch (e) {
    data.value = undefined
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('BAD_RESPONSE', String(e))
  }
  finally {
    loading.value = false
  }
}
const asText = () => (data.value?.hops ?? []).map(h => [`${h.method} ${h.url} → ${h.status} ${h.statusText}`, ...h.headers.map(x => `${x.name}: ${x.value}`), '', h.bodyPreview].join('\n')).join('\n\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[8rem_1fr]">
        <CSelect v-model:value="method" :label="t('tools.http-request.method')" :options="methodOptions" test-id="method" />
        <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com/api" :validation="urlValidation" monospace test-id="url" />
      </div>
      <CInputText v-model:value="headerText" :label="t('tools.http-request.headers')" placeholder="Accept: application/json" :validation="headerValidation" multiline :rows="3" monospace />
      <CInputText v-if="BODY_METHODS.includes(method)" v-model:value="body" :label="t('tools.http-request.body')" multiline :rows="4" monospace />
      <NFormItem :label="t('tools.http-request.follow')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="follow" :aria-label="t('tools.http-request.follow')" />
      </NFormItem>
      <p class="mt-0 text-xs nt-muted">
        {{ t('tools.http-request.privacy') }}
      </p>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!canRun" data-testid="run">
        {{ t('tools.http-request.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard v-for="(hop, i) in data.hops" :key="i" :title="data.hops.length > 1 ? t('tools.http-request.hop', { n: i + 1 }) : t('common.result')" data-testid="result">
      <HttpHopView :hop="hop" show-body />
    </CCard>
    <ResultActions :text="asText" :json="data" filename="http-request.json" />
  </template>
</template>
