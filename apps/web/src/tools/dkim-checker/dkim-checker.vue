<script setup lang="ts">
import type { EmailDkim } from './dkim-checker.service'
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
import { COMMON_SELECTORS, isValidSelector } from './dkim-checker.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const selector = useQueryParam({ name: 'selector', defaultValue: '' })
const domainV = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const selectorV = useValidation({ source: selector, rules: [{ validator: isValidSelector, message: t('tools.dkim-checker.invalidSelector') }] })
const canRun = computed(() => !!domain.value && !!selector.value && domainV.value.isValid && selectorV.value.isValid)
const { data, error, loading, run } = useApi<EmailDkim>('/v1/email/dkim')
const submit = () => canRun.value && run({ domain: domain.value, selector: selector.value })
onMounted(() => {
  if (canRun.value)
    void submit()
})
const rows = computed(() => (data.value?.record
  ? [
      { label: t('tools.dkim-checker.fields.name'), value: data.value.name, mono: true },
      { label: t('tools.dkim-checker.fields.key'), value: `${data.value.keyType.toUpperCase()}${data.value.keyBits ? ` ${data.value.keyBits} bit` : ''}`, noCopy: true },
      { label: t('tools.dkim-checker.fields.flags'), value: Object.entries(data.value.tags).filter(([k]) => k !== 'p').map(([k, v]) => `${k}=${v}`).join('; ') || '—', mono: true },
    ]
  : []))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="domainV" monospace test-id="domain" />
        <CInputText v-model:value="selector" :label="t('tools.dkim-checker.selector')" placeholder="selector1" :validation="selectorV" monospace test-id="selector" />
      </div>
      <div class="mb-3 flex flex-wrap gap-1">
        <NTag v-for="s in COMMON_SELECTORS" :key="s" size="small" class="cursor-pointer" :bordered="false" @click="selector = s">
          {{ s }}
        </NTag>
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!canRun" data-testid="run">
        {{ t('tools.dkim-checker.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <FindingsList :problems="data.problems" :warnings="data.warnings" :ok-text="t('tools.dkim-checker.ok')" />
    <CCard v-if="data.record" :title="t('common.result')" data-testid="result">
      <CKeyValueList :rows="rows" />
      <p class="mb-0 break-all text-xs nt-muted font-mono">
        {{ data.record }}
      </p>
      <div class="mt-3">
        <ResultActions :text="() => data!.record" :json="data" filename="dkim.json" />
      </div>
    </CCard>
  </template>
</template>
