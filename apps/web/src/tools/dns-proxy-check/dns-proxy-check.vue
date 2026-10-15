<script setup lang="ts">
import type { DnsLookup } from '../dns-lookup/dns-lookup.service'
import type { DnsVerdict, ProxyFinding } from './dns-proxy-check.service'
import type { EchoInfo } from '@/api/client'
import { NAlert, NButton } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiGet } from '@/api/client'
import ResultTable from '@/components/ResultTable.vue'
import { buildQuery } from '@/composables/useApi'
import CCard from '@/ui/c-card.vue'
import { lookupInBrowser } from '../dns-lookup/dns-lookup.service'
import { compareDns, proxyFindings, randomName, TEST_NAMES } from './dns-proxy-check.service'

const { t } = useI18n()
const running = ref(false)
const proxy = shallowRef<ProxyFinding[]>()
const proxyError = ref(false)
interface Row { name: string, server: string, pub: string, verdict: DnsVerdict }
const dnsRows = shallowRef<Row[]>([])
const dohBlocked = ref(false)
const ips = (l?: DnsLookup) => (l ? l.answer.filter(r => r.type === 'A').map(r => r.data).join(', ') || l.rcode : '—')
async function run() {
  running.value = true
  proxy.value = undefined
  proxyError.value = false
  dnsRows.value = []
  dohBlocked.value = false
  const marker = crypto.randomUUID()
  const withMarker: typeof fetch = (input, init) => fetch(input, { ...init, headers: { ...(init?.headers as Record<string, string>), 'X-NetTools-Probe': marker } })
  const echo = apiGet<EchoInfo>('/v1/echo', { fetch: withMarker }).then(e => (proxy.value = proxyFindings(e, marker, navigator.userAgent))).catch(() => (proxyError.value = true))
  const names = [...TEST_NAMES, randomName()]
  const rows = await Promise.all(names.map(async (name, i) => {
    const [server, pub] = await Promise.all([
      apiGet<DnsLookup>(`/v1/dns/lookup${buildQuery({ name, type: 'A', resolver: 'system' })}`).catch(() => undefined),
      lookupInBrowser(name, 'A', 'cloudflare').catch(() => {
        dohBlocked.value = true
        return undefined
      }),
    ])
    return { name, server: ips(server), pub: ips(pub), verdict: compareDns(server, pub, i === names.length - 1) }
  }))
  dnsRows.value = rows
  await echo
  running.value = false
}
const dnsVerdict = computed(() => {
  const v = dnsRows.value.map(r => r.verdict)
  if (!v.length)
    return undefined
  if (v.includes('nx-hijack'))
    return 'nx-hijack'
  if (v.includes('blocked'))
    return 'blocked'
  return v.includes('differ') ? 'differ' : 'clean'
})
const dnsType = { 'nx-hijack': 'error', 'blocked': 'warning', 'differ': 'info', 'clean': 'success' } as const
const proxyClean = computed(() => proxy.value?.length === 1 && proxy.value[0]!.id === 'clean')
const k = (key: string) => t(`tools.dns-proxy-check.columns.${key}`)
const tableRows = computed(() => dnsRows.value.map(r => ({ ...r, verdict: t(`tools.dns-proxy-check.verdicts.${r.verdict}`) })))
</script>

<template>
  <CCard>
    <NButton type="primary" :loading="running" data-testid="run" @click="run">
      {{ t('tools.dns-proxy-check.run') }}
    </NButton>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.dns-proxy-check.note') }}
    </p>
  </CCard>
  <CCard v-if="dnsRows.length" :title="t('tools.dns-proxy-check.dns')" data-testid="result">
    <NAlert v-if="dnsVerdict" :type="dnsType[dnsVerdict]" class="mb-3" data-testid="dns-verdict">
      {{ t(`tools.dns-proxy-check.dnsVerdict.${dnsVerdict}`) }}
    </NAlert>
    <NAlert v-if="dohBlocked" type="info" class="mb-3">
      {{ t('tools.dns-proxy-check.dohBlocked') }}
    </NAlert>
    <ResultTable :columns="[{ key: 'name', title: k('name'), mono: true, width: 240 }, { key: 'server', title: k('server'), mono: true, width: 200 }, { key: 'pub', title: k('pub'), mono: true, width: 200 }, { key: 'verdict', title: k('verdict'), width: 200 }]" :rows="tableRows" row-key="name" filename="dns-check.json" />
  </CCard>
  <CCard v-if="proxy || proxyError" :title="t('tools.dns-proxy-check.proxy')">
    <NAlert v-if="proxyError" type="warning">
      {{ t('tools.dns-proxy-check.proxyError') }}
    </NAlert>
    <NAlert v-else-if="proxyClean" type="success" data-testid="proxy-verdict">
      {{ t('tools.dns-proxy-check.findings.clean') }}
    </NAlert>
    <template v-else>
      <NAlert v-for="(f, i) in proxy" :key="i" :type="f.id === 'extra-hops' ? 'info' : 'warning'" class="mb-2" data-testid="proxy-verdict">
        {{ t(`tools.dns-proxy-check.findings.${f.id}`, { detail: f.detail }) }}
      </NAlert>
    </template>
  </CCard>
</template>
