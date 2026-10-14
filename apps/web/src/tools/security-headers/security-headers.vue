<script setup lang="ts">
import type { Check, SecurityHeaders } from './security-headers.service'
import { NButton, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHttpUrl } from '@/utils/url'
import { GRADE_TYPE, messageKey, sortChecks } from './security-headers.service'

const { t, te } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<SecurityHeaders>('/v1/http/security-headers')
const submit = () => url.value && validation.value.isValid && run({ url: url.value })
onMounted(() => {
  if (url.value)
    void submit()
})
const checks = computed(() => sortChecks(data.value?.checks ?? []))
const tagType = { fail: 'error', warn: 'warning', pass: 'success', info: 'default' } as const
function message(c: Check) {
  const k = messageKey(c)
  return te(k) ? t(k, { value: c.value, detail: c.detail }) : t(`tools.security-headers.status.${c.status}`)
}
const asText = () => [`${data.value?.grade} (${data.value?.score}/100) ${data.value?.url}`, ...checks.value.map(c => `[${c.status}] ${c.header}: ${message(c)}${c.value ? ` — ${c.value}` : ''}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.security-headers.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <div class="mb-4 flex flex-wrap items-center gap-3">
      <NTag :type="GRADE_TYPE[data.grade]" size="large" :bordered="false" class="text-2xl font-bold" data-testid="grade">
        {{ data.grade }}
      </NTag>
      <div>
        <div>{{ t('tools.security-headers.score', { score: data.score }) }}</div>
        <div class="break-all text-xs nt-muted font-mono">
          {{ data.status }} · {{ data.url }}
        </div>
      </div>
    </div>
    <ul class="m-0 list-none p-0">
      <li v-for="(c, i) in checks" :key="i" class="flex flex-wrap items-start gap-2 py-2" :data-testid="`check-${c.id}`">
        <NTag :type="tagType[c.status]" size="small" :bordered="false" class="mt-0.5 w-14 justify-center">
          {{ t(`tools.security-headers.status.${c.status}`) }}
        </NTag>
        <div class="min-w-0 flex-1">
          <div class="text-sm font-semibold">
            {{ c.header }}
          </div>
          <div class="text-sm">
            {{ message(c) }}
          </div>
          <div v-if="c.value" class="break-all text-xs nt-muted font-mono">
            {{ c.value }}
          </div>
        </div>
      </li>
    </ul>
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="security-headers.json" />
    </div>
  </CCard>
</template>
