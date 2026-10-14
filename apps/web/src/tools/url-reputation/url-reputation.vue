<script setup lang="ts">
import type { UrlReputation } from './url-reputation.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NTag } from 'naive-ui'
import { onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiGet } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { buildQuery } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { alertType, isUrlOrDomain, tagType } from './url-reputation.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: input, rules: [{ validator: isUrlOrDomain, message: t('tools.url-reputation.invalid') }] })
const result = shallowRef<UrlReputation>()
const error = ref<ApiErrorType>()
const running = ref(false)
async function run() {
  if (!input.value || !validation.value.isValid)
    return
  running.value = true
  error.value = undefined
  result.value = undefined
  try {
    result.value = await apiGet<UrlReputation>(`/v1/reputation/url${buildQuery({ url: input.value.trim() })}`)
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    running.value = false
  }
}
onMounted(() => {
  if (input.value)
    void run()
})
const detailRows = (d: Record<string, string>) => Object.entries(d).map(([k, v]) => ({ label: t(`tools.url-reputation.details.${k}`), value: v, mono: true }))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="input" :label="t('tools.url-reputation.input')" placeholder="https://example.com/login" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!input || !validation.isValid || running" data-testid="run">
        {{ t('tools.url-reputation.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.url-reputation.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="result">
    <NAlert :type="alertType(result.verdict)" class="mb-4" data-testid="result">
      {{ t(`tools.url-reputation.overall.${result.verdict}`, { domain: result.domain }) }}
    </NAlert>
    <CCard v-for="p in result.providers" :key="p.provider" :title="p.provider">
      <div class="mb-2 flex flex-wrap items-center gap-2">
        <NTag :type="tagType(p.verdict)" :bordered="false" size="small">
          {{ t(`tools.url-reputation.verdicts.${p.verdict}`) }}
        </NTag>
        <span v-if="p.summary" class="text-sm">{{ p.summary }}</span>
        <span v-if="p.error" class="text-sm">{{ p.error }}</span>
        <span v-if="p.verdict === 'skipped'" class="text-sm nt-muted">{{ t('tools.url-reputation.skipped') }}</span>
      </div>
      <CKeyValueList v-if="Object.keys(p.details).length" :rows="detailRows(p.details)" />
      <a v-if="p.link" :href="p.link" target="_blank" rel="noopener noreferrer" class="text-sm">{{ t('tools.url-reputation.open', { provider: p.provider }) }}</a>
    </CCard>
    <ResultActions :text="() => result!.providers.map(p => `${p.provider}: ${p.verdict}${p.summary ? ` — ${p.summary}` : ''}`).join('\n')" :json="result" filename="url-reputation.json" />
  </template>
</template>
