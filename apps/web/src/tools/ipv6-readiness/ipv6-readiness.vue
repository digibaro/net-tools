<script setup lang="ts">
import type { Ipv6Readiness } from './ipv6-readiness.service'
import { NAlert, NButton, NProgress, NTag } from 'naive-ui'
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
import { hostState } from './ipv6-readiness.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<Ipv6Readiness>('/v1/ipv6/readiness')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value.trim() })
onMounted(() => {
  if (domain.value)
    void submit()
})
const verdictType = { 'ready': 'success', 'partial': 'warning', 'not-ready': 'error' } as const
const stateType = { 'ok': 'success', 'no-aaaa': 'error', 'unreachable': 'warning', 'untested': 'info' } as const
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.ipv6-readiness.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="verdictType[data.verdict]" class="mb-4" data-testid="result">
      {{ t(`tools.ipv6-readiness.verdict.${data.verdict}`, { domain: data.domain }) }}
    </NAlert>
    <NAlert v-if="!data.serverIpv6" type="info" class="mb-4">
      {{ t('tools.ipv6-readiness.noServerV6') }}
      <RouterLink :to="{ path: '/ping', query: { target: data.domain, vantage: 'worldwide' } }">
        {{ t('tools.ipv6-readiness.tryWorldwide') }}
      </RouterLink>
    </NAlert>
    <CCard :title="t('tools.ipv6-readiness.score', { score: data.score })">
      <NProgress type="line" :percentage="data.score" :status="data.score >= 100 ? 'success' : data.score >= 50 ? 'warning' : 'error'" :show-indicator="false" :aria-label="t('tools.ipv6-readiness.score', { score: data.score })" class="mb-4 max-w-md" />
      <ul class="m-0 list-none p-0">
        <li v-for="h in data.hosts" :key="h.role + h.name" class="flex flex-wrap items-center gap-2 py-1 text-sm">
          <NTag size="small" :bordered="false" :type="stateType[hostState(h)]">
            {{ t(`tools.ipv6-readiness.state.${hostState(h)}`) }}
          </NTag>
          <span class="w-16 text-xs nt-muted uppercase">{{ t(`tools.ipv6-readiness.roles.${h.role}`) }}</span>
          <span class="font-mono">{{ h.name }}</span>
          <span v-if="h.aaaa.length" class="text-xs nt-muted font-mono">{{ h.aaaa.join(', ') }}</span>
          <span v-if="h.error" class="text-xs nt-muted">{{ h.error }}</span>
        </li>
      </ul>
      <div class="mt-3">
        <ResultActions :text="() => data!.hosts.map(h => `${h.role}\t${h.name}\t${hostState(h)}\t${h.aaaa.join(' ')}`).join('\n')" :json="data" filename="ipv6-readiness.json" />
      </div>
    </CCard>
  </template>
</template>
