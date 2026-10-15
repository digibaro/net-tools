<script setup lang="ts">
import type { SmtpBannerResult } from './smtp-banner-check.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { grade } from './smtp-banner-check.service'

const { t } = useI18n()
const host = useQueryParam({ name: 'host', defaultValue: '' })
const validation = useValidation({ source: host, rules: [{ validator: (s: string) => !s || isValidQueryName(s), message: t('tools.smtp-banner-check.invalid') }] })
const { data, error, loading, run } = useApi<SmtpBannerResult>('/v1/email/smtp-banner')
const submit = () => host.value && validation.value.isValid && run({ host: host.value.trim() })
onMounted(() => {
  if (host.value)
    void submit()
})
const g = computed(() => (data.value ? grade(data.value) : undefined))
const gradeType = { good: 'success', issues: 'warning', bad: 'error' } as const
const rows = computed(() => (data.value?.checks ?? []).map(c => ({ id: c.id, check: t(`tools.smtp-banner-check.checks.${c.id}`), result: c.pass ? '✓' : '✗', detail: c.detail || '—' })))
const k = (key: string) => t(`tools.smtp-banner-check.columns.${key}`)
const asText = () => [data.value?.probe.banner ?? '', ...rows.value.map(r => `${r.result} ${r.check}: ${r.detail}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="host" :label="t('tools.smtp-banner-check.host')" placeholder="mx.example.com" :validation="validation" monospace test-id="host" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!host || !validation.isValid" data-testid="run">
        {{ t('tools.smtp-banner-check.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data && g" :title="`${data.host} (${data.address})`" data-testid="result">
    <NAlert :type="gradeType[g]" class="mb-3" data-testid="verdict">
      {{ t(`tools.smtp-banner-check.grade.${g}`, { n: data.passed, total: data.checks.length }) }}
    </NAlert>
    <p v-if="data.via" class="nt-muted mt-0 text-sm">
      {{ t('tools.smtp-banner-check.via', { domain: data.via, host: data.host }) }}
    </p>
    <NAlert v-if="data.smtpBlocked" type="info" class="mb-3">
      {{ t('tools.smtp-banner-check.blocked') }}
    </NAlert>
    <pre v-if="data.probe.banner" class="m-0 mb-3 overflow-auto text-xs font-mono" tabindex="0">220 {{ data.probe.banner }}
250 {{ data.probe.greeting }}</pre>
    <ResultTable :columns="[{ key: 'result', title: '', width: 40 }, { key: 'check', title: k('check'), width: 300 }, { key: 'detail', title: k('detail'), mono: true, width: 340 }]" :rows="rows" row-key="id" :actions="false" :label="t('tools.smtp-banner-check.title')" />
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="smtp-banner.json" />
    </div>
  </CCard>
</template>
