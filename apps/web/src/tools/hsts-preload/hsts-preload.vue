<script setup lang="ts">
import type { HstsPreload } from './hsts-preload.service'
import { NAlert, NButton, NTag } from 'naive-ui'
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
import { statusType } from './hsts-preload.service'

const { t, te } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<HstsPreload>('/v1/http/hsts-preload')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value.trim() })
onMounted(() => {
  if (domain.value)
    void submit()
})
const status = (s: string) => (te(`tools.hsts-preload.status.${s}`) ? t(`tools.hsts-preload.status.${s}`) : s)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.hsts-preload.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="data.domain" data-testid="result">
    <template #header-extra>
      <NTag :type="statusType(data.status)" :bordered="false">
        {{ status(data.status) }}
      </NTag>
    </template>
    <p v-if="data.preloadedDomain && data.preloadedDomain !== data.domain" class="mt-0 text-sm">
      {{ t('tools.hsts-preload.viaParent', { parent: data.preloadedDomain }) }}
    </p>
    <NAlert v-if="data.checkError" type="info" class="mb-3">
      {{ t('tools.hsts-preload.checkError', { error: data.checkError }) }}
    </NAlert>
    <NAlert v-else :type="data.preloadable ? 'success' : 'warning'" class="mb-3" data-testid="preloadable">
      {{ data.preloadable ? t('tools.hsts-preload.meets') : t('tools.hsts-preload.fails', { n: data.errors.length }) }}
    </NAlert>
    <div class="flex flex-col gap-2">
      <NAlert v-for="e in data.errors" :key="e.code" type="error" :title="e.summary">
        {{ e.message }}
      </NAlert>
      <NAlert v-for="w in data.warnings" :key="w.code" type="warning" :title="w.summary">
        {{ w.message }}
      </NAlert>
    </div>
    <div class="mt-3">
      <ResultActions :text="() => [`${data!.domain}: ${data!.status}`, ...data!.errors.map(e => `error: ${e.summary}`), ...data!.warnings.map(w => `warning: ${w.summary}`)].join('\n')" :json="data" filename="hsts-preload.json" />
    </div>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.hsts-preload.source') }}
    </p>
  </CCard>
</template>
