<script setup lang="ts">
import type { HttpHops } from './redirect-tracer.service'
import { NAlert, NButton, NTimeline, NTimelineItem } from 'naive-ui'
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
import { isValidHttpUrl } from '@/utils/url'
import { summarize } from './redirect-tracer.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<HttpHops>('/v1/http/redirects')
const submit = () => url.value && validation.value.isValid && run({ url: url.value })
onMounted(() => {
  if (url.value)
    void submit()
})
const summary = computed(() => (data.value ? summarize(data.value.hops) : undefined))
const itemType = (status: number, err: string) => (err ? 'error' : status >= 400 ? 'warning' : status >= 300 ? 'info' : 'success')
const asText = () => (data.value?.hops ?? []).map((h, i) => `${i + 1}. ${h.status || h.errorCode} ${h.url}${h.location ? ` → ${h.location}` : ''}`).join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="http://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.redirect-tracer.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data && summary">
    <NAlert :type="summary.stoppedBy ? 'error' : summary.downgrade ? 'warning' : 'success'" class="mb-4" data-testid="summary">
      {{ summary.stoppedBy ? t('tools.redirect-tracer.stopped', { code: summary.stoppedBy }) : t('tools.redirect-tracer.summary', { redirects: summary.redirects, status: summary.finalStatus, url: summary.finalUrl }) }}
      <template v-if="summary.httpsUpgrade">
        {{ t('tools.redirect-tracer.upgrade') }}
      </template>
      <template v-if="summary.downgrade">
        {{ t('tools.redirect-tracer.downgrade') }}
      </template>
    </NAlert>
    <CCard :title="t('tools.redirect-tracer.chain')" data-testid="result">
      <NTimeline>
        <NTimelineItem v-for="(hop, i) in data.hops" :key="i" :type="itemType(hop.status, hop.errorCode)">
          <HttpHopView :hop="hop" :show-headers="false" />
        </NTimelineItem>
      </NTimeline>
      <ResultActions :text="asText" :json="data" filename="redirects.json" />
    </CCard>
  </template>
</template>
