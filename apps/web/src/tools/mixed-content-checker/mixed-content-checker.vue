<script setup lang="ts">
import type { MixedResult } from './mixed-content-checker.service'
import { NAlert, NButton, NSwitch } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHttpUrl } from '@/utils/url'
import { verdict } from './mixed-content-checker.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const showLinks = useQueryParam({ name: 'links', defaultValue: false })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<MixedResult>('/v1/http/mixed-content')
const submit = () => url.value && validation.value.isValid && run({ url: url.value.trim() })
onMounted(() => {
  if (url.value)
    void submit()
})
const v = computed(() => (data.value ? verdict(data.value) : undefined))
const alertType = { clean: 'success', upgraded: 'info', passive: 'warning', active: 'error' } as const
const rows = computed(() => (data.value?.items ?? [])
  .filter(i => showLinks.value || i.kind !== 'link')
  .map((i, n) => ({ id: String(n), kind: t(`tools.mixed-content-checker.kinds.${i.kind}`), element: `<${i.tag} ${i.attr}>`, url: i.url })))
const k = (key: string) => t(`tools.mixed-content-checker.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.mixed-content-checker.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.mixed-content-checker.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data && v" :title="t('tools.mixed-content-checker.summary', { url: data.url, resources: data.resources })" data-testid="result">
    <NAlert :type="alertType[v]" class="mb-3" data-testid="verdict">
      {{ t(`tools.mixed-content-checker.verdict.${v}`, { active: data.active, passive: data.passive, forms: data.forms }) }}
    </NAlert>
    <NAlert v-if="data.truncated" type="warning" class="mb-3">
      {{ t('tools.mixed-content-checker.truncated', { bytes: data.bodyBytes }) }}
    </NAlert>
    <label v-if="data.links" class="mb-3 flex items-center gap-2 text-sm">
      <NSwitch v-model:value="showLinks" size="small" :aria-label="t('tools.mixed-content-checker.showLinks', { n: data.links })" />
      {{ t('tools.mixed-content-checker.showLinks', { n: data.links }) }}
    </label>
    <ResultTable v-if="rows.length" :columns="[{ key: 'kind', title: k('kind'), width: 160 }, { key: 'element', title: k('element'), mono: true, width: 160 }, { key: 'url', title: 'URL', mono: true, width: 440 }]" :rows="rows" row-key="id" filename="mixed-content.json" />
  </CCard>
</template>
