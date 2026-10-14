<script setup lang="ts">
import type { EmailMx } from './mx-lookup.service'
import { NAlert, NButton, NFormItem, NSwitch, NTag } from 'naive-ui'
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
import { probeStatus } from './mx-lookup.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const probe = useQueryParam({ name: 'probe', defaultValue: true })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<EmailMx>('/v1/email/mx')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value, probe: probe.value ? undefined : false })
onMounted(() => {
  if (domain.value)
    void submit()
})
const statusType = { 'ok': 'success', 'no-starttls': 'warning', 'cert': 'warning', 'unreachable': 'error' } as const
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NFormItem :label="t('tools.mx-lookup.probe')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="probe" :aria-label="t('tools.mx-lookup.probe')" />
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.mx-lookup.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-if="data.nullMx" type="info" class="mb-4" data-testid="null-mx">
      {{ t('tools.mx-lookup.nullMx') }}
    </NAlert>
    <NAlert v-else-if="data.implicit" type="warning" class="mb-4">
      {{ t('tools.mx-lookup.implicit') }}
    </NAlert>
    <CCard v-for="h in data.hosts" :key="h.host" :title="`${h.preference} ${h.host}`" data-testid="result">
      <p class="mt-0 text-xs nt-muted font-mono">
        {{ h.addresses.join(', ') || h.error }}
      </p>
      <div v-for="p in h.probes" :key="p.address" class="mb-3">
        <div class="flex flex-wrap items-center gap-2">
          <NTag :type="statusType[probeStatus(p)]" size="small" :bordered="false">
            {{ t(`tools.mx-lookup.status.${probeStatus(p)}`) }}
          </NTag>
          <span class="text-sm font-mono">{{ p.address }}</span>
          <span v-if="p.connected" class="text-xs nt-muted">{{ p.connectMs }} ms</span>
        </div>
        <p v-if="p.banner" class="m-0 mt-1 text-xs font-mono">
          {{ p.banner }}
        </p>
        <p v-if="p.tls" class="m-0 mt-1 text-xs nt-muted">
          {{ p.tls.version }} · {{ p.tls.cipher }} · {{ p.tls.verified ? t('http.verified') : p.tls.verifyError }}
        </p>
        <p v-if="p.extensions.length" class="m-0 mt-1 text-xs nt-muted font-mono">
          {{ p.extensions.join(' · ') }}
        </p>
        <p v-if="p.error" class="m-0 mt-1 text-xs">
          {{ p.error }}
        </p>
      </div>
    </CCard>
    <ResultActions :text="() => data!.hosts.map(h => `${h.preference} ${h.host} ${h.addresses.join(' ')}`).join('\n')" :json="data" filename="mx.json" />
  </template>
</template>
