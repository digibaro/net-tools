<script setup lang="ts">
import type { DnsAxfr } from './zone-transfer-test.service'
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
import { serverState, typeSummary } from './zone-transfer-test.service'

const { t } = useI18n()
const zone = useQueryParam({ name: 'zone', defaultValue: '' })
const validation = useValidation({ source: zone, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<DnsAxfr>('/v1/dns/axfr')
const submit = () => zone.value && validation.value.isValid && run({ zone: zone.value.trim() })
onMounted(() => {
  if (zone.value)
    void submit()
})
const stateType = { open: 'error', refused: 'success', error: 'default' } as const
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="zone" :label="t('tools.zone-transfer-test.zone')" placeholder="example.com" :validation="validation" monospace test-id="zone" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!zone || !validation.isValid" data-testid="run">
        {{ t('tools.zone-transfer-test.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.anyAllowed ? 'error' : 'success'" class="mb-4" data-testid="result">
      {{ data.anyAllowed ? t('tools.zone-transfer-test.open') : t('tools.zone-transfer-test.closed', { n: data.nameservers.length }) }}
    </NAlert>
    <CCard v-for="s in data.servers" :key="`${s.name}${s.address}`" :title="s.name">
      <template #header-extra>
        <NTag :type="stateType[serverState(s)]" :bordered="false">
          {{ t(`tools.zone-transfer-test.state.${serverState(s)}`) }}
        </NTag>
      </template>
      <p class="mt-0 text-xs nt-muted font-mono">
        {{ s.address || '—' }}
      </p>
      <p v-if="s.error && !s.allowed" class="m-0 text-sm">
        {{ s.error }}
      </p>
      <template v-if="s.allowed">
        <p class="m-0 text-sm">
          {{ t('tools.zone-transfer-test.records', { n: s.records }) }}<span v-if="s.truncated"> ({{ t('tools.zone-transfer-test.truncated') }})</span> · {{ typeSummary(s.types) }}
        </p>
        <pre class="mb-0 max-h-60 overflow-auto text-xs font-mono" tabindex="0">{{ s.sample.map(r => `${r.name} ${r.ttl} ${r.type} ${r.data}`).join('\n') }}</pre>
      </template>
    </CCard>
    <ResultActions :text="() => data!.servers.map(s => `${s.name} ${s.address}: ${serverState(s)}${s.allowed ? ` (${s.records} records)` : ''}`).join('\n')" :json="data" filename="axfr.json" />
  </template>
</template>
