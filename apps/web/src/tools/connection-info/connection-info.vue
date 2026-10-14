<script setup lang="ts">
import type { EchoInfo } from '@/api/client'
import { NButton, NTag } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getEcho } from '@/api/client'
import ResultActions from '@/components/ResultActions.vue'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { fetchPublicIp } from '../what-is-my-ip/what-is-my-ip.service'
import { connectionEstimate, pageTiming, stack } from './connection-info.service'

const { t } = useI18n()
const loading = ref(false)
const v4 = ref<string>()
const v6 = ref<string>()
const echo = shallowRef<EchoInfo>()
const estimate = connectionEstimate()
const timing = pageTiming(performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined)
const online = ref(navigator.onLine)

async function refresh() {
  loading.value = true
  const [a, b, e] = await Promise.allSettled([fetchPublicIp(4), fetchPublicIp(6), getEcho()])
  v4.value = a.status === 'fulfilled' ? a.value : undefined
  v6.value = b.status === 'fulfilled' ? b.value : undefined
  echo.value = e.status === 'fulfilled' ? e.value : undefined
  online.value = navigator.onLine
  loading.value = false
}
onMounted(refresh)

const k = (key: string) => t(`tools.connection-info.fields.${key}`)
const yesNo = (b: boolean) => t(b ? 'common.yes' : 'common.no')
const ipRows = computed(() => [
  { label: 'IPv4', value: v4.value ?? t('tools.connection-info.noConnectivity'), mono: !!v4.value },
  { label: 'IPv6', value: v6.value ?? t('tools.connection-info.noConnectivity'), mono: !!v6.value },
  ...(echo.value ? [{ label: k('toServer'), value: `IPv${echo.value.family} · ${echo.value.protocol}`, noCopy: true }] : []),
])
const stackType = computed(() => stack(v4.value, v6.value))
const estimateRows = computed(() => (estimate
  ? [
      { label: k('type'), value: estimate.type ?? '—', noCopy: true },
      { label: k('effectiveType'), value: estimate.effectiveType ?? '—', noCopy: true },
      { label: k('downlink'), value: estimate.downlinkMbps !== undefined ? `≈ ${estimate.downlinkMbps} Mbit/s` : '—', noCopy: true },
      { label: k('rtt'), value: estimate.rttMs !== undefined ? `≈ ${estimate.rttMs} ms` : '—', noCopy: true },
      { label: k('saveData'), value: estimate.saveData === undefined ? '—' : yesNo(estimate.saveData), noCopy: true },
    ]
  : []))
const timingRows = computed(() => (timing
  ? [
      { label: k('protocol'), value: timing.protocol || '—', noCopy: true },
      { label: k('dns'), value: `${timing.dnsMs} ms`, noCopy: true },
      { label: k('connect'), value: `${timing.connectMs} ms`, noCopy: true },
      { label: k('tls'), value: `${timing.tlsMs} ms`, noCopy: true },
      { label: k('ttfb'), value: `${timing.ttfbMs} ms`, noCopy: true },
      { label: k('download'), value: `${timing.downloadMs} ms`, noCopy: true },
    ]
  : []))
const browserRows = computed(() => [
  { label: k('online'), value: yesNo(online.value), noCopy: true },
  { label: k('secure'), value: yesNo(window.isSecureContext), noCopy: true },
  { label: k('doh'), value: t('tools.connection-info.dohNote'), noCopy: true },
])
const stackTagType = { 'dual': 'success', 'v4-only': 'warning', 'v6-only': 'info', 'none': 'error' } as const
</script>

<template>
  <CCard :title="t('tools.connection-info.ipTitle')" data-testid="result">
    <template #header-extra>
      <NTag v-if="!loading" :type="stackTagType[stackType]" :bordered="false" data-testid="stack">
        {{ t(`tools.connection-info.stack.${stackType}`) }}
      </NTag>
    </template>
    <CKeyValueList :rows="ipRows" />
    <div class="mt-3 flex flex-wrap gap-2">
      <NButton size="small" :loading="loading" @click="refresh">
        {{ t('tools.connection-info.refresh') }}
      </NButton>
      <ResultActions :text="() => [...ipRows, ...estimateRows, ...timingRows].map(r => `${r.label}: ${r.value}`).join('\n')" :json="{ ipv4: v4, ipv6: v6, server: echo, estimate, timing }" filename="connection-info.json" />
    </div>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.connection-info.ipNote') }}
    </p>
  </CCard>
  <CCard :title="t('tools.connection-info.estimateTitle')">
    <CKeyValueList v-if="estimateRows.length" :rows="estimateRows" />
    <p v-else class="m-0 text-sm nt-muted">
      {{ t('tools.connection-info.noEstimate') }}
    </p>
  </CCard>
  <CCard v-if="timingRows.length" :title="t('tools.connection-info.timingTitle')">
    <CKeyValueList :rows="timingRows" />
  </CCard>
  <CCard :title="t('tools.connection-info.browserTitle')">
    <CKeyValueList :rows="browserRows" />
  </CCard>
</template>
