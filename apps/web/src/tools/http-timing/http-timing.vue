<script setup lang="ts">
import type { HttpHops } from './http-timing.service'
import { NAlert, NButton, NFormItem, NSwitch } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import HttpHopView from '@/components/HttpHopView.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHttpUrl } from '@/utils/url'
import { phases, ttfbRating } from './http-timing.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const follow = useQueryParam({ name: 'follow', defaultValue: false })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<HttpHops>('/v1/http/timing')
const submit = () => url.value && validation.value.isValid && run({ url: url.value, follow: follow.value || undefined })
onMounted(() => {
  if (url.value)
    void submit()
})
const last = computed(() => data.value?.hops.at(-1))
const rows = computed(() => (last.value ? phases(last.value.timing).map(p => ({ phase: t(`http.phases.${p.id}`), ms: `${p.ms} ms`, start: `${p.startMs} ms` })) : []))
const asText = () => [...rows.value.map(r => `${r.phase}\t${r.ms}\t(start ${r.start})`), `${t('http.total')}\t${last.value?.timing.totalMs} ms`].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NFormItem :label="t('tools.http-request.follow')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="follow" :aria-label="t('tools.http-request.follow')" />
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.http-timing.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="last">
    <NAlert v-if="!last.error" :type="ttfbRating(last.timing.ttfbMs) === 'slow' ? 'warning' : 'success'" class="mb-4" data-testid="rating">
      {{ t(`tools.http-timing.rating.${ttfbRating(last.timing.ttfbMs)}`, { ttfb: last.timing.ttfbMs, total: last.timing.totalMs }) }}
    </NAlert>
    <CCard :title="t('tools.http-timing.breakdown')" data-testid="result">
      <ResultTable :columns="[{ key: 'phase', title: t('tools.http-timing.phase'), width: 140 }, { key: 'ms', title: t('tools.http-timing.duration'), width: 120 }, { key: 'start', title: t('tools.http-timing.start'), width: 120 }]" :rows="rows" :actions="false" :label="t('tools.http-timing.breakdown')" />
      <div class="mt-4">
        <HttpHopView :hop="last" :show-headers="false" />
      </div>
      <ResultActions :text="asText" :json="data" filename="http-timing.json" />
    </CCard>
  </template>
</template>
