<script setup lang="ts">
import type { components } from '@/api/schema'
import { NCollapse, NCollapseItem, NTag, useThemeVars } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import ResultTable from './ResultTable.vue'

type Hop = components['schemas']['HttpHop']
const props = withDefaults(defineProps<{ hop: Hop, showBody?: boolean, showHeaders?: boolean }>(), { showBody: false, showHeaders: true })
const { t } = useI18n()
const theme = useThemeVars()

const statusType = computed(() => {
  const s = props.hop.status
  return s >= 500 ? 'error' : s >= 400 ? 'warning' : s >= 300 ? 'info' : 'success'
})
const phases = computed(() => {
  const x = props.hop.timing
  return [
    { id: 'resolve', ms: x.resolveMs, color: theme.value.infoColor },
    { id: 'connect', ms: x.connectMs, color: theme.value.warningColor },
    { id: 'tls', ms: x.tlsMs, color: theme.value.primaryColor },
    { id: 'ttfb', ms: Math.max(0, x.ttfbMs - x.connectMs - x.tlsMs), color: theme.value.successColor },
    { id: 'transfer', ms: x.transferMs, color: theme.value.textColor3 },
  ].filter(p => p.ms > 0)
})
const total = computed(() => phases.value.reduce((s, p) => s + p.ms, 0) || 1)
const tlsRows = computed(() => props.hop.tls
  ? [
      { label: t('http.tls'), value: `${props.hop.tls.version} · ${props.hop.tls.cipher}${props.hop.tls.alpn ? ` · ${props.hop.tls.alpn}` : ''}`, noCopy: true },
      { label: t('http.certificate'), value: `${props.hop.tls.subject} (${t('http.until', { date: props.hop.tls.notAfter.slice(0, 10) })})`, mono: true },
      { label: t('http.verification'), value: props.hop.tls.verifyError || t('http.verified'), noCopy: true },
    ]
  : [])
</script>

<template>
  <div>
    <div class="mb-2 flex flex-wrap items-center gap-2">
      <NTag v-if="!hop.error" :type="statusType" :bordered="false" data-testid="status">
        {{ hop.status }} {{ hop.statusText }}
      </NTag>
      <NTag v-else type="error" :bordered="false">
        {{ hop.errorCode }}
      </NTag>
      <span class="break-all text-sm font-mono">{{ hop.method }} {{ hop.url }}</span>
    </div>
    <p v-if="hop.error" class="m-0 text-sm" data-testid="hop-error">
      {{ hop.error }}
    </p>
    <template v-else>
      <p class="m-0 mb-2 text-xs nt-muted">
        {{ hop.remoteAddr }} · {{ hop.proto }}<template v-if="hop.location">
          · {{ t('http.location') }}: <span class="font-mono">{{ hop.location }}</span>
        </template>
      </p>
      <div class="mb-1 h-3 flex overflow-hidden rounded" role="img" :aria-label="t('http.timingBar', { total: hop.timing.totalMs })">
        <div v-for="p in phases" :key="p.id" :style="{ width: `${(p.ms / total) * 100}%`, background: p.color }" :title="`${t(`http.phases.${p.id}`)}: ${p.ms} ms`" />
      </div>
      <p class="m-0 mb-3 text-xs nt-muted">
        <span v-for="p in phases" :key="p.id" class="mr-3">{{ t(`http.phases.${p.id}`) }} {{ p.ms }} ms</span>
        <span>· {{ t('http.total') }} {{ hop.timing.totalMs }} ms</span>
      </p>
      <CKeyValueList v-if="tlsRows.length" :rows="tlsRows" class="mb-3" />
      <NCollapse v-if="showHeaders || (showBody && hop.bodyPreview)" :default-expanded-names="showHeaders ? ['headers'] : []">
        <NCollapseItem v-if="showHeaders" :title="t('http.headers', { count: hop.headers.length })" name="headers">
          <ResultTable :columns="[{ key: 'name', title: t('http.name'), mono: true, width: 220 }, { key: 'value', title: t('http.value'), mono: true, width: 420 }]" :rows="hop.headers" :actions="false" :label="t('http.headers', { count: hop.headers.length })" />
        </NCollapseItem>
        <NCollapseItem v-if="showBody && hop.bodyPreview" :title="t('http.body', { bytes: hop.bodyBytes })" name="body">
          <pre class="m-0 max-h-96 overflow-auto whitespace-pre-wrap break-all text-xs font-mono" tabindex="0" data-testid="body">{{ hop.bodyPreview }}</pre>
          <p v-if="hop.bodyTruncated" class="mb-0 text-xs nt-muted">
            {{ t('http.truncated') }}
          </p>
        </NCollapseItem>
      </NCollapse>
    </template>
  </div>
</template>
