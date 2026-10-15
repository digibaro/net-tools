<script setup lang="ts">
import type { CdnResult } from './cdn-waf-detector.service'
import { NAlert, NButton, NTag } from 'naive-ui'
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
import { isValidHttpUrl } from '@/utils/url'
import { sortDetections } from './cdn-waf-detector.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<CdnResult>('/v1/http/cdn')
const submit = () => url.value && validation.value.isValid && run({ url: url.value.trim() })
onMounted(() => {
  if (url.value)
    void submit()
})
const detections = computed(() => sortDetections(data.value?.detections ?? []))
const kindType = { cdn: 'success', waf: 'warning', bot: 'warning', hosting: 'info', cache: 'default', tls: 'default' } as const
const k = (key: string) => t(`tools.cdn-waf-detector.fields.${key}`)
const rows = computed(() => data.value
  ? [
      { label: k('url'), value: `${data.value.url} (HTTP ${data.value.status})`, mono: true },
      { label: k('address'), value: data.value.remoteAddr || '—', mono: true },
      { label: k('cnames'), value: data.value.cnames.join(' → ') || '—', mono: true },
      { label: k('server'), value: data.value.server || '—', mono: true },
      { label: k('cacheStatus'), value: data.value.cacheStatus || '—', mono: true },
      { label: k('issuer'), value: data.value.issuer || '—', noCopy: true },
    ]
  : [])
const asText = () => [...detections.value.map(d => `${d.name} (${d.kind}): ${d.evidence.join('; ')}`), ...rows.value.map(r => `${r.label}: ${r.value}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.cdn-waf-detector.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <NAlert v-if="!detections.length" type="info" class="mb-3">
      {{ t('tools.cdn-waf-detector.none') }}
    </NAlert>
    <div v-for="d in detections" :key="d.name" class="mb-3" data-testid="detection">
      <div class="flex flex-wrap items-center gap-2">
        <span class="font-semibold">{{ d.name }}</span>
        <NTag size="small" :bordered="false" :type="kindType[d.kind]">
          {{ t(`tools.cdn-waf-detector.kinds.${d.kind}`) }}
        </NTag>
      </div>
      <ul class="m-0 mt-1 pl-5 text-sm font-mono">
        <li v-for="e in d.evidence" :key="e" class="break-all">
          {{ e }}
        </li>
      </ul>
    </div>
    <CKeyValueList :rows="rows" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.cdn-waf-detector.note') }}
    </p>
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="cdn-waf.json" />
    </div>
  </CCard>
</template>
