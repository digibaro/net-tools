<script setup lang="ts">
import type { CacheResult } from './cache-header-analyzer.service'
import { NAlert, NButton } from 'naive-ui'
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
import { duration } from './cache-header-analyzer.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<CacheResult>('/v1/http/cache')
const submit = () => url.value && validation.value.isValid && run({ url: url.value.trim() })
onMounted(() => {
  if (url.value)
    void submit()
})
const k = (key: string) => t(`tools.cache-header-analyzer.fields.${key}`)
const ttl = (cacheable: boolean, s: number) => (!cacheable ? t('tools.cache-header-analyzer.notCached') : duration(s) ?? t('tools.cache-header-analyzer.unknown'))
const rows = computed(() => {
  const d = data.value
  if (!d)
    return []
  return [
    { label: k('url'), value: `${d.url} (HTTP ${d.status})`, mono: true },
    { label: 'Cache-Control', value: d.cacheControl || '—', mono: true },
    { label: k('browser'), value: ttl(d.browserCacheable, d.browserTtl), noCopy: true },
    { label: k('shared'), value: ttl(d.sharedCacheable, d.sharedTtl), noCopy: true },
    { label: k('source'), value: t(`tools.cache-header-analyzer.sources.${d.ttlSource.replace('+', '_')}`), noCopy: true },
    { label: 'ETag', value: d.etag || '—', mono: true },
    { label: 'Last-Modified', value: d.lastModified || '—', mono: true },
    { label: 'Expires', value: d.expires || '—', mono: true },
    { label: 'Vary', value: d.vary || '—', mono: true },
    { label: 'Age', value: d.age || '—', mono: true },
    { label: k('cacheStatus'), value: d.cacheStatus || '—', mono: true },
    ...(d.repeat ? [{ label: k('repeat'), value: t('tools.cache-header-analyzer.repeatValue', { sent: d.repeat.sent || t('tools.cache-header-analyzer.noValidator'), status: d.repeat.status, cache: d.repeat.cacheStatus || '—' }), noCopy: true }] : []),
  ]
})
const severity = { info: 'info', warn: 'warning', bad: 'error' } as const
const findings = computed(() => (data.value?.findings ?? []).map(f => ({ severity: severity[f.severity], text: t(`tools.cache-header-analyzer.findings.${f.id}`, { detail: f.detail }) })))
const asText = () => [...rows.value.map(r => `${r.label}: ${r.value}`), ...findings.value.map(f => `- ${f.text}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com/style.css" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.cache-header-analyzer.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <NAlert v-for="(f, i) in findings" :key="i" :type="f.severity" class="mb-2" data-testid="finding">
      {{ f.text }}
    </NAlert>
    <CKeyValueList :rows="rows" />
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="cache-headers.json" />
    </div>
  </CCard>
</template>
