<script setup lang="ts">
import type { NdpHost, NdpRouter } from './ipv6-neighbors.service'
import { NAlert, NButton, NFormItem, NInputNumber } from 'naive-ui'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isInterfaceName } from '../lldp-neighbors/lldp-neighbors.service'
import { addressMode, lifetime } from './ipv6-neighbors.service'

const { t } = useI18n()
const iface = useQueryParam({ name: 'interface', defaultValue: '' })
const seconds = useQueryParam({ name: 'seconds', defaultValue: 4 })
const ifaceV = useValidation({ source: iface, rules: [{ validator: isInterfaceName, message: t('tools.ipv6-neighbors.invalidInterface') }] })
const hosts = ref<NdpHost[]>([])
const routers = ref<NdpRouter[]>([])
const source = ref('')
const { status, error, start } = useSse('/v1/lan/ndp/stream', {
  meta: d => (source.value = (d as { source: string }).source),
  host: d => hosts.value.push(d as NdpHost),
  router: d => routers.value.push(d as NdpRouter),
  done: (d) => {
    const s = (d as { summary: { hosts: NdpHost[] } }).summary
    hosts.value = s.hosts // includes neighbours that answered the echo but not the solicitation
  },
})
const running = computed(() => status.value === 'running')
function run() {
  if (!ifaceV.value.isValid)
    return
  hosts.value = []
  routers.value = []
  start({ interface: iface.value.trim() || undefined, seconds: seconds.value })
}
const hostRows = computed(() => hosts.value.map(h => ({ ip: h.ip, mac: h.mac || '—', role: h.router ? t('tools.ipv6-neighbors.router') : t('tools.ipv6-neighbors.host') })))
function routerRows(r: NdpRouter) {
  const k = (key: string) => t(`tools.ipv6-neighbors.ra.${key}`)
  return [
    { label: k('router'), value: r.mac ? `${r.ip} (${r.mac})` : r.ip, mono: true },
    { label: k('mode'), value: t(`tools.ipv6-neighbors.modes.${addressMode(r)}`), noCopy: true },
    { label: k('prefixes'), value: r.prefixes.map(p => `${p.prefix}${p.slaac ? ' SLAAC' : ''} (${lifetime(p.validSeconds)})`).join(', ') || '—', mono: true },
    { label: k('dns'), value: r.dns.join(', ') || '—', mono: true },
    { label: k('search'), value: r.search.join(', ') || '—', mono: true },
    { label: k('mtu'), value: r.mtu ? String(r.mtu) : '—', mono: true },
    { label: k('lifetime'), value: r.lifetimeSeconds ? lifetime(r.lifetimeSeconds) : t('tools.ipv6-neighbors.notDefault'), noCopy: true },
    { label: k('preference'), value: r.preference, noCopy: true },
  ]
}
const c = (k: string) => t(`tools.ipv6-neighbors.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="iface" :label="t('tools.ipv6-neighbors.interface')" placeholder="eth0" :validation="ifaceV" monospace test-id="interface" />
        <NFormItem :label="t('tools.ipv6-neighbors.seconds')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="seconds" :min="2" :max="10" :show-button="false" :input-props="{ 'aria-label': t('tools.ipv6-neighbors.seconds') }" />
        </NFormItem>
      </div>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="running || !ifaceV.isValid" data-testid="run">
        {{ t('tools.ipv6-neighbors.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.ipv6-neighbors.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="running || status === 'done'">
    <CCard v-for="r in routers" :key="r.ip" :title="t('tools.ipv6-neighbors.routerTitle', { ip: r.ip })" data-testid="router">
      <CKeyValueList :rows="routerRows(r)" />
    </CCard>
    <NAlert v-if="status === 'done' && !routers.length" type="info" class="mb-4">
      {{ t('tools.ipv6-neighbors.noRouter') }}
    </NAlert>
    <CCard :title="t('tools.ipv6-neighbors.found', { n: hosts.length })" data-testid="result">
      <p class="mt-0 text-sm nt-muted">
        {{ t('tools.ipv6-neighbors.from', { source }) }}
      </p>
      <ResultTable v-if="hostRows.length" :columns="[{ key: 'ip', title: c('ip'), mono: true, width: 260 }, { key: 'mac', title: c('mac'), mono: true, width: 170 }, { key: 'role', title: c('role'), width: 100 }]" :rows="hostRows" row-key="ip" filename="ipv6-neighbors.json" />
    </CCard>
  </template>
</template>
