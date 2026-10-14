<script setup lang="ts">
import type { WellKnown } from './well-known-files.service'
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
import { fileState } from './well-known-files.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<WellKnown>('/v1/http/wellknown')
const submit = () => url.value && validation.value.isValid && run({ url: url.value })
onMounted(() => {
  if (url.value)
    void submit()
})
const stateType = { found: 'success', missing: 'default', error: 'warning' } as const
const securityRows = computed(() => Object.entries(data.value?.security.fields ?? {}).map(([k, v]) => ({ label: k, value: v.join('\n'), mono: true })))
const finding = (code: string) => t(`tools.well-known-files.findings.${code}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('tools.well-known-files.site')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.well-known-files.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard title="robots.txt" data-testid="result">
      <template #header-extra>
        <NTag size="small" :bordered="false" :type="stateType[fileState(data.robots.file)]">
          {{ t(`tools.well-known-files.state.${fileState(data.robots.file)}`, { status: data.robots.file.status }) }}
        </NTag>
      </template>
      <NAlert v-if="data.robots.blocksAll" type="warning" class="mb-3">
        {{ t('tools.well-known-files.blocksAll') }}
      </NAlert>
      <ul v-if="data.robots.groups.length" class="m-0 pl-5 text-sm">
        <li v-for="(g, i) in data.robots.groups" :key="i">
          <span class="font-mono">{{ g.agents.join(', ') }}</span>: {{ t('tools.well-known-files.group', { disallow: g.disallow, allow: g.allow }) }}<template v-if="g.crawlDelay">
            · Crawl-delay {{ g.crawlDelay }}
          </template>
        </li>
      </ul>
      <p v-if="data.robots.sitemaps.length" class="mb-0 mt-2 break-all text-xs nt-muted font-mono">
        Sitemap: {{ data.robots.sitemaps.join(', ') }}
      </p>
    </CCard>
    <CCard title="security.txt">
      <template #header-extra>
        <NTag size="small" :bordered="false" :type="stateType[fileState(data.security.file)]">
          {{ t(`tools.well-known-files.state.${fileState(data.security.file)}`, { status: data.security.file.status }) }}
        </NTag>
      </template>
      <template v-if="data.security.file.found">
        <p class="mt-0 break-all text-xs nt-muted font-mono">
          {{ data.security.file.url }}
        </p>
        <CKeyValueList :rows="securityRows" />
        <ul v-if="data.security.problems.length || data.security.warnings.length" class="mb-0 mt-2 pl-5 text-sm">
          <li v-for="p in data.security.problems" :key="p">
            <strong>{{ finding(p) }}</strong>
          </li>
          <li v-for="w in data.security.warnings" :key="w">
            {{ finding(w) }}
          </li>
        </ul>
      </template>
      <p v-else class="m-0 text-sm">
        {{ t('tools.well-known-files.noSecurityTxt') }}
      </p>
    </CCard>
    <CCard :title="t('tools.well-known-files.sitemap')">
      <template #header-extra>
        <NTag size="small" :bordered="false" :type="stateType[fileState(data.sitemap.file)]">
          {{ t(`tools.well-known-files.state.${fileState(data.sitemap.file)}`, { status: data.sitemap.file.status }) }}
        </NTag>
      </template>
      <p class="mt-0 break-all text-xs nt-muted font-mono">
        {{ data.sitemap.file.url }}
      </p>
      <p v-if="data.sitemap.file.found" class="m-0 text-sm">
        {{ t(data.sitemap.kind === 'sitemapindex' ? 'tools.well-known-files.sitemapIndex' : 'tools.well-known-files.sitemapUrls', { n: data.sitemap.entries }) }}<template v-if="data.sitemap.newest">
          · {{ t('tools.well-known-files.newest', { date: data.sitemap.newest }) }}
        </template><template v-if="data.sitemap.capped">
          · {{ t('tools.well-known-files.capped') }}
        </template>
      </p>
    </CCard>
    <ResultActions :text="() => [`robots.txt: ${fileState(data!.robots.file)}`, `security.txt: ${fileState(data!.security.file)} ${data!.security.problems.join(', ')}`, `sitemap: ${fileState(data!.sitemap.file)} ${data!.sitemap.entries}`].join('\n')" :json="data" filename="well-known.json" />
  </template>
</template>
