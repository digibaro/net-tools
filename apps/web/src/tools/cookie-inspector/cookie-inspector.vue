<script setup lang="ts">
import type { Cookies } from './cookie-inspector.service'
import { NAlert, NButton, NTag } from 'naive-ui'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHttpUrl } from '@/utils/url'
import { attributes, severity } from './cookie-inspector.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<Cookies>('/v1/http/cookies')
const submit = () => url.value && validation.value.isValid && run({ url: url.value })
onMounted(() => {
  if (url.value)
    void submit()
})
const finding = (code: string) => t(`tools.cookie-inspector.findings.${code}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.cookie-inspector.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-if="!data.cookies.length" type="info" class="mb-4" data-testid="result">
      {{ t('tools.cookie-inspector.none', { url: data.url }) }}
    </NAlert>
    <template v-else>
      <p class="text-sm nt-muted" data-testid="result">
        {{ t('tools.cookie-inspector.count', { n: data.cookies.length }) }}
      </p>
      <CCard v-for="(c, i) in data.cookies" :key="i" :title="c.name" data-testid="cookie">
        <template #header-extra>
          <NTag size="small" :bordered="false" :type="severity(c)">
            {{ t(`tools.cookie-inspector.severity.${severity(c)}`) }}
          </NTag>
        </template>
        <p class="m-0 text-sm font-mono">
          {{ attributes(c) || '—' }}
        </p>
        <p class="m-0 mt-1 text-xs nt-muted">
          {{ t('tools.cookie-inspector.value', { value: c.value, n: c.valueLength }) }} ·
          {{ c.session ? t('tools.cookie-inspector.session') : c.expires ? t('tools.cookie-inspector.expires', { date: c.expires.slice(0, 10) }) : '' }} ·
          {{ t('tools.cookie-inspector.setBy', { url: c.setBy }) }}
        </p>
        <ul v-if="c.problems.length || c.warnings.length" class="mb-0 mt-2 pl-5 text-sm">
          <li v-for="p in c.problems" :key="p">
            <strong>{{ finding(p) }}</strong>
          </li>
          <li v-for="w in c.warnings" :key="w">
            {{ finding(w) }}
          </li>
        </ul>
      </CCard>
      <ResultActions :text="() => data!.cookies.map(c => `${c.name}: ${attributes(c)} [${[...c.problems, ...c.warnings].join(', ')}]`).join('\n')" :json="data" filename="cookies.json" />
    </template>
  </template>
</template>
