<script setup lang="ts">
import type { DnsWildcard } from './wildcard-detector.service'
import { NAlert, NButton } from 'naive-ui'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { wildcardAnswers } from './wildcard-detector.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<DnsWildcard>('/v1/dns/wildcard')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value.trim() })
onMounted(() => {
  if (domain.value)
    void submit()
})
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.wildcard-detector.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.wildcard ? 'warning' : 'success'" class="mb-4" data-testid="result">
      {{ data.wildcard ? t(data.consistent ? 'tools.wildcard-detector.found' : 'tools.wildcard-detector.foundVarying', { domain: data.domain, types: data.types.join(', ') }) : t('tools.wildcard-detector.none', { domain: data.domain }) }}
    </NAlert>
    <CCard v-if="data.wildcard" :title="t('tools.wildcard-detector.answers')">
      <pre class="m-0 overflow-auto text-sm font-mono" tabindex="0">{{ wildcardAnswers(data).join('\n') }}</pre>
    </CCard>
    <CCard :title="t('tools.wildcard-detector.probes')">
      <ul class="m-0 list-none p-0 text-sm font-mono">
        <li v-for="p in data.probes" :key="p.name" class="py-0.5">
          {{ p.name }} → {{ p.answers.length ? p.answers.map(a => `${a.type} ${a.data}`).join(', ') : p.rcode }}
        </li>
      </ul>
      <div class="mt-3">
        <ResultActions :text="() => wildcardAnswers(data!).join('\n') || t('tools.wildcard-detector.none', { domain: data!.domain })" :json="data" filename="wildcard.json" />
      </div>
    </CCard>
  </template>
</template>
