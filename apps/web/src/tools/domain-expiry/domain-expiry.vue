<script setup lang="ts">
import type { DomainExpiry } from './domain-expiry.service'
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
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { day, urgency } from './domain-expiry.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<DomainExpiry>('/v1/domain/expiry')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value.trim() })
onMounted(() => {
  if (domain.value)
    void submit()
})
const level = computed(() => urgency(data.value?.daysLeft ?? null))
const tagType = { expired: 'error', soon: 'warning', ok: 'success', unknown: 'default' } as const
const rows = computed(() => {
  const d = data.value
  if (!d)
    return []
  const k = (key: string) => t(`registry.fields.${key}`)
  return [
    { label: k('expires'), value: d.expires ? day(d.expires) : '—', mono: true },
    { label: k('created'), value: d.registered ? day(d.registered) : '—', mono: true },
    { label: k('updated'), value: d.updated ? day(d.updated) : '—', mono: true },
    { label: k('registrar'), value: d.registrar || '—' },
    { label: k('status'), value: d.status.join(', ') || '—' },
  ]
})
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.domain-expiry.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="data.domain" data-testid="result">
    <template #header-extra>
      <NTag :type="tagType[level]" :bordered="false" data-testid="days-left">
        {{ level === 'unknown' ? t('tools.domain-expiry.unknown') : level === 'expired' ? t('tools.domain-expiry.expired') : t('tools.domain-expiry.daysLeft', { n: data.daysLeft }) }}
      </NTag>
    </template>
    <CKeyValueList :rows="rows" />
    <div class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" filename="domain-expiry.json" />
    </div>
  </CCard>
</template>
