<script setup lang="ts">
import type { IcmpType } from './protocol-reference.service'
import { NTabPane, NTabs } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { filterIcmp, filterProtocols, ICMP } from './protocol-reference.service'

const { t } = useI18n()
const query = useQueryParam({ name: 'q', defaultValue: '' })
const tab = useQueryParam({ name: 'tab', defaultValue: 'ip' })
const rfcUrl = (rfc: string | undefined) => (rfc ? `https://www.rfc-editor.org/rfc/rfc${rfc.slice(4)}` : '')
const k = (key: string) => t(`tools.protocol-reference.columns.${key}`)

const protocolRows = computed(() => filterProtocols(query.value).map(p => ({
  number: p.number,
  keyword: p.keyword,
  name: p.name,
  ext: p.ipv6ExtensionHeader ? t('common.yes') : '',
  reference: p.rfc[0] ?? '',
  url: rfcUrl(p.rfc[0]),
})))
const protocolColumns = computed(() => [
  { key: 'number', title: k('number'), width: 80, sortable: true },
  { key: 'keyword', title: k('keyword'), width: 120, mono: true },
  { key: 'name', title: k('name'), width: 300 },
  { key: 'ext', title: k('ext'), width: 120 },
  { key: 'reference', title: k('reference'), width: 110, hrefKey: 'url' },
])

function icmpRows(list: IcmpType[]) {
  return filterIcmp(query.value, list).flatMap(ty => [
    { type: ty.type, code: '', name: ty.name, reference: ty.rfc[0] ?? '', url: rfcUrl(ty.rfc[0]) },
    ...ty.codes.map(c => ({ type: '', code: c.code, name: `↳ ${c.name}`, reference: '', url: '' })),
  ])
}
const icmpColumns = computed(() => [
  { key: 'type', title: k('type'), width: 70 },
  { key: 'code', title: k('code'), width: 70 },
  { key: 'name', title: k('name'), width: 380 },
  { key: 'reference', title: k('reference'), width: 110, hrefKey: 'url' },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="query" :label="t('tools.protocol-reference.search')" placeholder="6, tcp, unreachable…" test-id="input" />
  </CCard>
  <CCard data-testid="result">
    <NTabs v-model:value="tab" type="line" animated>
      <NTabPane name="ip" :tab="t('tools.protocol-reference.tabs.ip')">
        <ResultTable :columns="protocolColumns" :rows="protocolRows" :max-height="520" :label="t('tools.protocol-reference.tabs.ip')" filename="ip-protocols.json" />
      </NTabPane>
      <NTabPane name="icmp" :tab="t('tools.protocol-reference.tabs.icmp')">
        <ResultTable :columns="icmpColumns" :rows="icmpRows(ICMP.v4)" :max-height="520" :label="t('tools.protocol-reference.tabs.icmp')" filename="icmp-types.json" />
      </NTabPane>
      <NTabPane name="icmpv6" :tab="t('tools.protocol-reference.tabs.icmpv6')">
        <ResultTable :columns="icmpColumns" :rows="icmpRows(ICMP.v6)" :max-height="520" :label="t('tools.protocol-reference.tabs.icmpv6')" filename="icmpv6-types.json" />
      </NTabPane>
    </NTabs>
  </CCard>
  <p class="text-xs nt-muted">
    {{ t('tools.protocol-reference.source') }}
  </p>
</template>
