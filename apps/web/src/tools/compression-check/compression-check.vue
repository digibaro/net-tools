<script setup lang="ts">
import type { Compression } from './compression-check.service'
import { NAlert, NButton } from 'naive-ui'
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
import { saving, size } from './compression-check.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isValidHttpUrl, message: t('http.invalidUrl') }] })
const { data, error, loading, run } = useApi<Compression>('/v1/http/compression')
const submit = () => url.value && validation.value.isValid && run({ url: url.value })
onMounted(() => {
  if (url.value)
    void submit()
})
const rows = computed(() => (data.value?.results ?? []).map(r => ({
  requested: r.requested,
  served: r.error ? '—' : r.served,
  supported: r.requested === 'identity' ? '—' : r.supported ? t('common.yes') : t('common.no'),
  size: r.error ? r.error : `${size(r.bytes)}${r.truncated ? ' +' : ''}`,
  saving: r.supported ? saving(r.ratio) : '',
})))
const c = (key: string) => t(`tools.compression-check.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.compression-check.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.best ? 'success' : 'warning'" class="mb-4" data-testid="result">
      {{ data.best ? t('tools.compression-check.best', { enc: data.best }) : t('tools.compression-check.none') }}
    </NAlert>
    <CCard :title="data.url">
      <p class="mt-0 text-xs nt-muted">
        {{ data.contentType || '—' }} · Vary: {{ data.vary || '—' }}
      </p>
      <ResultTable :columns="[{ key: 'requested', title: c('requested'), mono: true, width: 110 }, { key: 'served', title: c('served'), mono: true, width: 110 }, { key: 'supported', title: c('supported'), width: 100 }, { key: 'size', title: c('size'), width: 160 }, { key: 'saving', title: c('saving'), width: 90 }]" :rows="rows" row-key="requested" filename="compression.json" />
      <p v-if="data.vary && !/accept-encoding/i.test(data.vary) && data.best" class="mb-0 text-sm">
        {{ t('tools.compression-check.varyMissing') }}
      </p>
    </CCard>
  </template>
</template>
