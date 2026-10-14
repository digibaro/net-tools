<script setup lang="ts">
import type { Geo } from './ip-geolocation.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getEcho } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { flag, isIpAddress, mapUrl } from './ip-geolocation.service'

const { t } = useI18n()
const ip = useQueryParam({ name: 'ip', defaultValue: '' })
// Optional: force "local" or "ipinfo"; by default the server picks what is available.
const source = useQueryParam({ name: 'source', defaultValue: '' })
const validation = useValidation({ source: ip, rules: [{ validator: isIpAddress, message: t('tools.reverse-dns.invalid') }] })
const { data, error, loading, run } = useApi<Geo>('/v1/geo')
const submit = () => ip.value && validation.value.isValid && run({ ip: ip.value.trim(), source: source.value || undefined })
onMounted(async () => {
  if (!ip.value) {
    // Start with the visitor's own address as the server sees it.
    try {
      ip.value = (await getEcho()).ip
    }
    catch {
      return
    }
  }
  void submit()
})
const rows = computed(() => {
  const d = data.value
  if (!d?.found)
    return []
  const k = (key: string) => t(`tools.ip-geolocation.fields.${key}`)
  return [
    { label: k('country'), value: `${flag(d.countryCode)} ${d.country} (${d.countryCode})`.trim() },
    { label: k('region'), value: d.region },
    { label: k('city'), value: d.city },
    { label: k('postal'), value: d.postal, mono: true },
    { label: k('continent'), value: d.continent },
    { label: k('coordinates'), value: d.latitude !== null && d.longitude !== null ? `${d.latitude}, ${d.longitude}` : '', mono: true },
    { label: k('accuracy'), value: d.accuracyKm ? `± ${d.accuracyKm} km` : '', noCopy: true },
    { label: k('timeZone'), value: d.timeZone, mono: true },
    { label: k('asn'), value: d.asn ? `AS${d.asn} ${d.asOrg}` : '' },
    { label: k('prefix'), value: d.prefix, mono: true },
  ].filter(r => r.value)
})
const asnOnly = computed(() => data.value && !data.value.found && data.value.asn)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="ip" :label="t('tools.reverse-dns.ip')" placeholder="1.1.1.1" :validation="validation" monospace test-id="ip" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ip || !validation.isValid" data-testid="run">
        {{ t('tools.ip-geolocation.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-if="!data.public" type="info" class="mb-4" data-testid="not-public">
      {{ t('tools.ip-geolocation.notPublic', { ip: data.ip }) }}
    </NAlert>
    <NAlert v-else-if="!data.found" type="warning" class="mb-4" data-testid="not-found">
      {{ t('tools.ip-geolocation.notFound', { ip: data.ip }) }}
      <span v-if="asnOnly"> AS{{ data.asn }} {{ data.asOrg }}</span>
    </NAlert>
    <CCard v-if="data.found" :title="data.ip" data-testid="result">
      <CKeyValueList :rows="rows" />
      <div class="mt-3 flex flex-wrap items-center gap-3">
        <NButton v-if="data.latitude !== null && data.longitude !== null" tag="a" :href="mapUrl(data.latitude, data.longitude)" target="_blank" rel="noopener noreferrer" size="small" secondary>
          {{ t('tools.ip-geolocation.openMap') }}
        </NButton>
        <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" filename="geolocation.json" />
      </div>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.ip-geolocation.source', { db: data.database }) }}
      </p>
    </CCard>
  </template>
</template>
