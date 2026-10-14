<script setup lang="ts">
import type { DnsLookup, DohProvider } from './dns-lookup.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NButton, NFormItem, NInputNumber, NRadioButton, NRadioGroup, NSwitch } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiGet } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import DnsAnswer from '@/components/DnsAnswer.vue'
import RemoteResults from '@/components/RemoteResults.vue'
import { buildQuery } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useRemote } from '@/composables/useRemote'
import { useValidation } from '@/composables/useValidation'
import { useVantage } from '@/composables/useVantage'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { isValidQueryName, lookupInBrowser, RECORD_TYPES, RESOLVER_PRESETS, TRANSPORTS } from './dns-lookup.service'

const { t } = useI18n()
const caps = useCapabilitiesStore()
const name = useQueryParam({ name: 'name', defaultValue: '' })
const type = useQueryParam({ name: 'type', defaultValue: 'A' })
const vantage = useQueryParam({ name: 'from', defaultValue: 'server' })
const resolver = useQueryParam({ name: 'resolver', defaultValue: 'system' })
const custom = useQueryParam({ name: 'custom', defaultValue: '' })
const transport = useQueryParam({ name: 'transport', defaultValue: 'udp' })
const dnssec = useQueryParam({ name: 'dnssec', defaultValue: false })
const doh = useQueryParam({ name: 'doh', defaultValue: 'cloudflare' })

const serverAvailable = computed(() => caps.state !== 'unreachable' && caps.capabilities?.tools['dns.lookup']?.enabled !== false)
const fromServer = computed(() => vantage.value === 'server' && serverAvailable.value)
// Worldwide shares the remote locations/limit params with the other Globalping-capable tools.
const { locations, limit, available: remoteAvailable } = useVantage()
const worldwide = computed(() => vantage.value === 'worldwide' && remoteAvailable.value)
const remote = useRemote()
const REMOTE_DNS_TYPES = ['A', 'AAAA', 'CNAME', 'MX', 'NS', 'TXT', 'SOA', 'CAA', 'PTR', 'SRV', 'HTTPS']

const validation = useValidation({ source: name, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const typeOptions = RECORD_TYPES.map(v => ({ label: v, value: v }))
const resolverOptions = computed(() => [...RESOLVER_PRESETS, 'custom'].map(v => ({ label: t(`tools.dns-lookup.resolvers.${v}`), value: v })))
const transportOptions = computed(() => TRANSPORTS.filter(tr => resolver.value !== 'system' || tr === 'udp' || tr === 'tcp').map(v => ({ label: t(`tools.dns-lookup.transports.${v}`), value: v })))
const dohOptions = [{ label: 'Cloudflare', value: 'cloudflare' }, { label: 'Google', value: 'google' }]

const loading = ref(false)
const result = shallowRef<DnsLookup>()
const error = shallowRef<ApiErrorType>()

async function run() {
  if (!name.value.trim() || !validation.value.isValid)
    return
  if (worldwide.value) {
    remote.start({ type: 'dns', target: name.value.trim(), dnsType: REMOTE_DNS_TYPES.includes(type.value) ? type.value : 'A', locations: locations.value, limit: limit.value })
    return
  }
  loading.value = true
  error.value = undefined
  try {
    result.value = fromServer.value
      ? await apiGet<DnsLookup>(`/v1/dns/lookup${buildQuery({
          name: name.value,
          type: type.value,
          resolver: resolver.value === 'custom' ? custom.value : resolver.value,
          transport: resolver.value === 'system' && !['udp', 'tcp'].includes(transport.value) ? 'udp' : transport.value,
          dnssec: dnssec.value || undefined,
        })}`)
      : await lookupInBrowser(name.value, type.value, doh.value as DohProvider)
  }
  catch (e) {
    result.value = undefined
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('PROVIDER_ERROR', e instanceof Error ? e.message : String(e))
  }
  finally {
    loading.value = false
  }
}

onMounted(() => {
  if (name.value)
    void run()
})
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_9rem]">
        <CInputText v-model:value="name" :label="t('tools.dns-lookup.name')" placeholder="example.com" :validation="validation" monospace test-id="name" />
        <CSelect v-model:value="type" :label="t('tools.dns-lookup.type')" :options="typeOptions" filterable test-id="type" />
      </div>
      <NFormItem :label="t('tools.dns-lookup.vantage')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="vantage" name="vantage">
          <NRadioButton value="server" :disabled="!serverAvailable" :label="t('tools.dns-lookup.fromServer')" />
          <NRadioButton value="browser" :label="t('tools.dns-lookup.fromBrowser')" />
          <NRadioButton v-if="remoteAvailable" value="worldwide" :label="t('remote.worldwide')" data-testid="vantage-worldwide" />
        </NRadioGroup>
      </NFormItem>
      <div v-if="worldwide" class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="locations" :label="t('remote.locations')" :placeholder="t('remote.locationsPlaceholder')" test-id="locations" />
        <NFormItem :label="t('remote.limit')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="limit" :min="1" :max="10" :show-button="false" :input-props="{ 'aria-label': t('remote.limit') }" />
        </NFormItem>
      </div>
      <div v-if="fromServer" class="grid gap-x-4 sm:grid-cols-3">
        <CSelect v-model:value="resolver" :label="t('tools.dns-lookup.resolver')" :options="resolverOptions" test-id="resolver" />
        <CSelect v-model:value="transport" :label="t('tools.dns-lookup.transport')" :options="transportOptions" />
        <NFormItem :label="t('tools.dns-lookup.dnssec')" :show-feedback="false" class="mb-3">
          <NSwitch v-model:value="dnssec" :aria-label="t('tools.dns-lookup.dnssec')" />
        </NFormItem>
        <div v-if="resolver === 'custom'" class="sm:col-span-3">
          <CInputText v-model:value="custom" :label="t('tools.dns-lookup.custom')" placeholder="9.9.9.9, dns.example:853 or https://dns.example/dns-query" monospace test-id="custom" />
        </div>
      </div>
      <div v-else-if="!worldwide" class="max-w-60">
        <CSelect v-model:value="doh" :label="t('tools.dns-lookup.dohProvider')" :options="dohOptions" />
        <p class="mt-0 text-xs nt-muted">
          {{ t('tools.dns-lookup.browserNote') }}
        </p>
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading || remote.status.value === 'running'" :disabled="!name.trim() || !validation.isValid" data-testid="run">
        {{ t('tools.dns-lookup.run') }}
      </NButton>
    </form>
  </CCard>
  <RemoteResults v-if="worldwide" :meta="remote.meta.value" :probes="remote.probes.value" :summary="remote.summary.value" :status="remote.status.value" :error="remote.error.value" />
  <template v-else>
    <ApiError v-if="error" :error="error" />
    <CCard v-if="result" :title="t('common.result')" data-testid="result">
      <DnsAnswer :result="result" filename="dns-lookup.json" />
    </CCard>
  </template>
</template>
