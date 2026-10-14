<script setup lang="ts">
import type { EmailMtaSts } from './mta-sts-checker.service'
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
import { sectionState } from './mta-sts-checker.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<EmailMtaSts>('/v1/email/mtasts')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value.trim() })
onMounted(() => {
  if (domain.value)
    void submit()
})
const stateType = { absent: 'default', ok: 'success', warn: 'warning', fail: 'error' } as const
const stsRows = computed(() => {
  const s = data.value?.mtaSts
  if (!s?.record)
    return []
  const k = (key: string) => t(`tools.mta-sts-checker.fields.${key}`)
  return [
    { label: k('record'), value: s.record, mono: true },
    { label: k('mode'), value: s.mode || '—', noCopy: true },
    { label: k('maxAge'), value: s.maxAge ? `${s.maxAge} s` : '—', noCopy: true },
    { label: k('mx'), value: s.mx.join(', ') || '—', mono: true },
    { label: k('policy'), value: s.policyUrl, mono: true },
  ]
})
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.mta-sts-checker.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard title="MTA-STS" data-testid="result">
      <template #header-extra>
        <NTag size="small" :bordered="false" :type="stateType[sectionState(data.mtaSts)]">
          {{ t(`tools.mta-sts-checker.state.${sectionState(data.mtaSts)}`) }}
        </NTag>
      </template>
      <FindingsList :problems="data.mtaSts.problems" :warnings="data.mtaSts.warnings" :ok-text="t('tools.mta-sts-checker.stsOk')" />
      <CKeyValueList :rows="stsRows" />
      <p v-if="data.mtaSts.unmatched.length" class="mb-0 text-sm">
        {{ t('tools.mta-sts-checker.unmatched', { list: data.mtaSts.unmatched.join(', ') }) }}
      </p>
    </CCard>
    <CCard title="TLS-RPT">
      <template #header-extra>
        <NTag size="small" :bordered="false" :type="stateType[sectionState(data.tlsRpt)]">
          {{ t(`tools.mta-sts-checker.state.${sectionState(data.tlsRpt)}`) }}
        </NTag>
      </template>
      <FindingsList :problems="data.tlsRpt.problems" :warnings="[]" :ok-text="t('tools.mta-sts-checker.rptOk', { list: data.tlsRpt.rua.join(', ') })" />
    </CCard>
    <CCard title="BIMI">
      <template #header-extra>
        <NTag size="small" :bordered="false" :type="stateType[sectionState(data.bimi)]">
          {{ t(`tools.mta-sts-checker.state.${sectionState(data.bimi)}`) }}
        </NTag>
      </template>
      <FindingsList :problems="data.bimi.problems" :warnings="data.bimi.warnings" :ok-text="t('tools.mta-sts-checker.bimiOk')" />
      <CKeyValueList v-if="data.bimi.record" :rows="[{ label: t('tools.mta-sts-checker.fields.logo'), value: data.bimi.logo || '—', mono: true }, { label: t('tools.mta-sts-checker.fields.vmc'), value: data.bimi.certificate || '—', mono: true }]" />
    </CCard>
    <ResultActions :text="() => [`MTA-STS: ${sectionState(data!.mtaSts)} ${data!.mtaSts.problems.join(', ')}`, `TLS-RPT: ${sectionState(data!.tlsRpt)}`, `BIMI: ${sectionState(data!.bimi)} ${data!.bimi.problems.join(', ')}`].join('\n')" :json="data" filename="mta-sts.json" />
  </template>
</template>
