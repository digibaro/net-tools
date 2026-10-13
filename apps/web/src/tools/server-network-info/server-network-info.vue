<script setup lang="ts">
import type { ServerNetinfo } from './server-network-info.service'
import { NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { sortInterfaces } from './server-network-info.service'

const { t } = useI18n()
const caps = useCapabilitiesStore()
const enabled = computed(() => caps.capabilities?.tools['net.serverinfo']?.enabled === true)
const { data, error, loading, run } = useApi<ServerNetinfo>('/v1/server/netinfo')
onMounted(async () => {
  await caps.load()
  if (enabled.value)
    void run()
})
const k = (key: string) => t(`tools.server-network-info.columns.${key}`)
const ifaces = computed(() => sortInterfaces(data.value?.interfaces ?? []).map(i => ({ name: i.name, addresses: i.addresses.join('\n') || '—', mtu: i.mtu, mac: i.mac || '—', flags: i.flags.join(' ') })))
const routes = computed(() => (data.value?.routes ?? []).map(r => ({ destination: r.destination, gateway: r.gateway || t('tools.server-network-info.direct'), interface: r.interface, metric: r.metric })))
const summary = computed(() => data.value
  ? [
      { label: k('hostname'), value: data.value.hostname, mono: true },
      { label: k('gateways'), value: data.value.gateways.join(', ') || '—', mono: true },
      { label: k('resolvers'), value: data.value.resolvers.nameservers.join(', ') || '—', mono: true },
      { label: k('search'), value: data.value.resolvers.search.join(' ') || '—', mono: true },
      { label: k('lanMode'), value: t(data.value.lanMode ? 'common.on' : 'common.off'), noCopy: true },
    ]
  : [])
</script>

<template>
  <div v-if="enabled" class="mb-4 flex justify-end">
    <NButton :loading="loading" @click="run()">
      {{ t('tools.what-is-my-ip.refresh') }}
    </NButton>
  </div>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="t('tools.server-network-info.summary')" data-testid="result">
      <CKeyValueList :rows="summary" />
      <div class="mt-3">
        <ResultActions :text="() => JSON.stringify(data, null, 2)" :json="data" filename="server-netinfo.json" />
      </div>
    </CCard>
    <CCard :title="t('tools.server-network-info.interfaces')">
      <ResultTable :columns="[{ key: 'name', title: k('name'), mono: true, width: 110 }, { key: 'addresses', title: k('addresses'), mono: true, width: 300 }, { key: 'mtu', title: 'MTU', width: 80 }, { key: 'mac', title: 'MAC', mono: true, width: 160 }, { key: 'flags', title: k('flags'), width: 220 }]" :rows="ifaces" :actions="false" class="whitespace-pre-line" :label="t('tools.server-network-info.interfaces')" />
    </CCard>
    <CCard :title="t('tools.server-network-info.routes')">
      <ResultTable :columns="[{ key: 'destination', title: k('destination'), mono: true, width: 220 }, { key: 'gateway', title: k('gateway'), mono: true, width: 200 }, { key: 'interface', title: k('interface'), width: 110 }, { key: 'metric', title: k('metric'), width: 90 }]" :rows="routes" :actions="false" :label="t('tools.server-network-info.routes')" />
    </CCard>
  </template>
</template>
