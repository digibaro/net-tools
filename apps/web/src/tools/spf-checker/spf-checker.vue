<script setup lang="ts">
import type { EmailSpf } from './spf-checker.service'
import { NButton, NProgress, NTag } from 'naive-ui'
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
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { flatten } from './spf-checker.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<EmailSpf>('/v1/email/spf')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value })
onMounted(() => {
  if (domain.value)
    void submit()
})
const tree = computed(() => (data.value?.record ? flatten(data.value.record) : []))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.spf-checker.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <FindingsList :problems="data.problems" :warnings="data.warnings" :ok-text="t('tools.spf-checker.ok')" />
    <CCard :title="t('common.result')" data-testid="result">
      <p v-if="data.record.record" class="mt-0 break-all text-sm font-mono" data-testid="record">
        {{ data.record.record }}
      </p>
      <div class="mb-3 max-w-md">
        <div class="mb-1 text-sm">
          {{ t('tools.spf-checker.lookups', { n: data.lookups }) }}
        </div>
        <NProgress type="line" :percentage="Math.min(100, data.lookups * 10)" :status="data.lookups > 10 ? 'error' : data.lookups > 8 ? 'warning' : 'success'" :show-indicator="false" :aria-label="t('tools.spf-checker.lookups', { n: data.lookups })" />
      </div>
      <h3 class="mb-2 text-sm font-semibold">
        {{ t('tools.spf-checker.tree') }}
      </h3>
      <ul class="m-0 list-none p-0 text-sm font-mono">
        <li v-for="(l, i) in tree" :key="i" :style="{ paddingLeft: `${l.depth * 1.25}rem` }" class="py-0.5">
          {{ l.text }}
          <NTag v-if="l.lookup" size="tiny" :bordered="false" class="ml-1">
            DNS
          </NTag>
          <span v-if="l.error" class="ml-1">({{ t(`email.findings.${l.error}`) }})</span>
        </li>
      </ul>
      <div class="mt-3">
        <ResultActions :text="() => tree.map(l => '  '.repeat(l.depth) + l.text).join('\n')" :json="data" filename="spf.json" />
      </div>
    </CCard>
  </template>
</template>
