<script setup lang="ts">
import type { PeeringDbNetwork } from './peeringdb-lookup.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { parseAsn, portSpeed } from './peeringdb-lookup.service'

const { t } = useI18n()
const asn = useQueryParam({ name: 'asn', defaultValue: '' })
const validation = useValidation({ source: asn, rules: [{ validator: (v: string) => parseAsn(v) !== undefined, message: t('bgp.invalidAsn') }] })
const { data, error, loading, run } = useApi<PeeringDbNetwork>('/v1/peeringdb')
const submit = () => parseAsn(asn.value) !== undefined && run({}, `/AS${parseAsn(asn.value)}`)
onMounted(() => {
  if (asn.value)
    void submit()
})
const rows = computed(() => {
  const d = data.value
  if (!d?.found)
    return []
  const k = (key: string) => t(`tools.peeringdb-lookup.fields.${key}`)
  return [
    { label: k('name'), value: d.aka ? `${d.name} (${d.aka})` : d.name },
    { label: k('type'), value: [d.type, d.scope, d.ratio, d.traffic].filter(Boolean).join(' · ') || '—', noCopy: true },
    { label: k('policy'), value: [d.policy, d.policyLocations, d.policyContracts].filter(Boolean).join(' · ') || '—', noCopy: true },
    { label: k('policyUrl'), value: d.policyUrl || '—', mono: true },
    { label: k('asSet'), value: d.irrAsSet || '—', mono: true },
    { label: k('prefixes'), value: t('tools.peeringdb-lookup.maxPrefixes', { v4: d.prefixes4, v6: d.prefixes6 }), noCopy: true },
    { label: k('website'), value: d.website || '—', mono: true },
    { label: k('lookingGlass'), value: d.lookingGlass || '—', mono: true },
    { label: k('updated'), value: d.updated, mono: true },
  ]
})
const ixRows = computed(() => (data.value?.ixs ?? []).map((x, i) => ({ id: i, name: x.name, speed: portSpeed(x.speedMbps), ipv4: x.ipv4 || '—', ipv6: x.ipv6 || '—', rs: x.rsPeer ? t('common.yes') : t('common.no'), link: `https://www.peeringdb.com/ix/${x.ixId}` })))
const facRows = computed(() => (data.value?.facilities ?? []).map(f => ({ name: f.name, city: f.city, country: f.country, link: `https://www.peeringdb.com/fac/${f.facId}` })))
const c = (key: string) => t(`tools.peeringdb-lookup.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="asn" :label="t('bgp.asn')" placeholder="AS13335" :validation="validation" monospace test-id="asn" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="parseAsn(asn) === undefined" data-testid="run">
        {{ t('bgp.lookUp') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-if="!data.found" type="info" class="mb-4" data-testid="not-found">
      {{ t('tools.peeringdb-lookup.notFound', { asn: data.asn }) }}
    </NAlert>
    <template v-else>
      <CCard :title="`AS${data.asn}`" data-testid="result">
        <CKeyValueList :rows="rows" />
        <div class="mt-3 flex flex-wrap gap-2">
          <NButton tag="a" :href="data.peeringdbUrl" target="_blank" rel="noopener noreferrer" size="small" secondary>
            {{ t('tools.peeringdb-lookup.open') }}
          </NButton>
          <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" :filename="`peeringdb-as${data.asn}.json`" />
        </div>
      </CCard>
      <CCard v-if="ixRows.length" :title="t('tools.peeringdb-lookup.ixs', { n: data.ixCount })">
        <ResultTable :columns="[{ key: 'name', title: c('ix'), hrefKey: 'link', width: 280 }, { key: 'speed', title: c('speed'), width: 80 }, { key: 'ipv4', title: 'IPv4', mono: true, width: 150 }, { key: 'ipv6', title: 'IPv6', mono: true, width: 220 }, { key: 'rs', title: c('rs'), width: 110 }]" :rows="ixRows" row-key="id" :max-height="400" :filename="`peeringdb-as${data.asn}-ix.json`" />
      </CCard>
      <CCard v-if="facRows.length" :title="t('tools.peeringdb-lookup.facilities', { n: data.facilityCount })">
        <ResultTable :columns="[{ key: 'name', title: c('facility'), hrefKey: 'link', width: 320 }, { key: 'city', title: c('city'), width: 140 }, { key: 'country', title: c('country'), width: 90 }]" :rows="facRows" row-key="name" :max-height="400" :filename="`peeringdb-as${data.asn}-fac.json`" />
      </CCard>
      <CCard v-if="data.contacts.length" :title="t('tools.peeringdb-lookup.contacts')">
        <CKeyValueList :rows="data.contacts.map(p => ({ label: p.role, value: [p.name, p.email, p.url].filter(Boolean).join(' · ') }))" />
      </CCard>
    </template>
  </template>
</template>
