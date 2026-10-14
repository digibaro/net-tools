<script setup lang="ts">
import type { CloudRanges } from './cloud-ip-lookup.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { failedSources, isIpAddress } from './cloud-ip-lookup.service'

const { t } = useI18n()
const ip = useQueryParam({ name: 'ip', defaultValue: '' })
const validation = useValidation({ source: ip, rules: [{ validator: isIpAddress, message: t('tools.reverse-dns.invalid') }] })
const { data, error, loading, run } = useApi<CloudRanges>('/v1/cloud-ranges')
const submit = () => ip.value && validation.value.isValid && run({ ip: ip.value.trim() })
onMounted(() => {
  if (ip.value)
    void submit()
})
const rows = computed(() => (data.value?.matches ?? []).map(m => ({ ...m, service: m.service || '—', region: m.region || '—' })))
const failed = computed(() => (data.value ? failedSources(data.value) : []))
const c = (key: string) => t(`tools.cloud-ip-lookup.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="ip" :label="t('tools.reverse-dns.ip')" placeholder="52.95.110.1" :validation="validation" monospace test-id="ip" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ip || !validation.isValid" data-testid="run">
        {{ t('bgp.lookUp') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.provider ? 'success' : 'info'" class="mb-4" data-testid="result">
      {{ data.provider ? t('tools.cloud-ip-lookup.found', { ip: data.ip, provider: data.provider }) : t('tools.cloud-ip-lookup.none', { ip: data.ip }) }}
    </NAlert>
    <CCard v-if="rows.length" :title="t('tools.cloud-ip-lookup.matches')">
      <ResultTable :columns="[{ key: 'provider', title: c('provider'), width: 180 }, { key: 'prefix', title: c('prefix'), mono: true, width: 190 }, { key: 'service', title: c('service'), width: 180 }, { key: 'region', title: c('region'), width: 180 }]" :rows="rows" row-key="prefix" filename="cloud-ranges.json" />
    </CCard>
    <p class="text-xs nt-muted">
      {{ t('tools.cloud-ip-lookup.sources', { list: data.sources.map(s => s.name).join(', ') }) }}
      <template v-if="failed.length">
        {{ t('tools.cloud-ip-lookup.failed', { list: failed.join(', ') }) }}
      </template>
    </p>
  </template>
</template>
