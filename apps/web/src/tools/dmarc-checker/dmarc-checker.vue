<script setup lang="ts">
import type { EmailDmarc } from './dmarc-checker.service'
import { NButton, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import FindingsList from '@/components/FindingsList.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { strength } from './dmarc-checker.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<EmailDmarc>('/v1/email/dmarc')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value })
onMounted(() => {
  if (domain.value)
    void submit()
})
const rows = computed(() => {
  const d = data.value
  if (!d || !d.record)
    return []
  const k = (key: string) => t(`tools.dmarc-checker.fields.${key}`)
  const uris = (list: EmailDmarc['rua']) => list.map(u => `${u.uri}${u.external ? (u.authorized ? ' ✓' : ' ✕') : ''}`).join(', ') || '—'
  return [
    { label: k('foundAt'), value: `_dmarc.${d.foundAt}`, mono: true },
    { label: k('policy'), value: `${d.policy} (${t('tools.dmarc-checker.subdomains')}: ${d.subPolicy})`, noCopy: true },
    { label: k('pct'), value: `${d.pct} %`, noCopy: true },
    { label: k('alignment'), value: `DKIM ${d.adkim === 's' ? 'strict' : 'relaxed'}, SPF ${d.aspf === 's' ? 'strict' : 'relaxed'}`, noCopy: true },
    { label: k('rua'), value: uris(d.rua), mono: true },
    { label: k('ruf'), value: uris(d.ruf), mono: true },
  ]
})
const strengthType = { strong: 'success', weak: 'warning', none: 'error' } as const
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.dmarc-checker.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <FindingsList :problems="data.problems" :warnings="data.warnings" :ok-text="t('tools.dmarc-checker.ok')" />
    <CCard v-if="data.record" :title="t('common.result')" data-testid="result">
      <template #header-extra>
        <NTag :type="strengthType[strength(data.policy, data.pct)]" :bordered="false">
          {{ t(`tools.dmarc-checker.strength.${strength(data.policy, data.pct)}`) }}
        </NTag>
      </template>
      <p class="mt-0 break-all text-sm font-mono">
        {{ data.record }}
      </p>
      <CKeyValueList :rows="rows" />
      <div class="mt-3">
        <ResultActions :text="() => data!.record" :json="data" filename="dmarc.json" />
      </div>
    </CCard>
  </template>
</template>
