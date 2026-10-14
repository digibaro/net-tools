<script setup lang="ts">
import type { components } from '@/api/schema'
import { NTag } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import ResultTable from './ResultTable.vue'

type DnsLookup = components['schemas']['DnsLookup']
const props = defineProps<{ result: DnsLookup, filename?: string }>()
const { t } = useI18n()

const rcodeType = computed(() => (props.result.rcode === 'NOERROR' ? 'success' : props.result.rcode === 'NXDOMAIN' ? 'warning' : 'error'))
const flags = computed(() => Object.entries(props.result.flags).filter(([, v]) => v).map(([k]) => k.toUpperCase()).join(' ') || '—')
const summary = computed(() => [
  { label: t('dns.resolver'), value: `${props.result.resolver} · ${props.result.transport.toUpperCase()}`, noCopy: true },
  { label: t('dns.flags'), value: flags.value, mono: true, noCopy: true },
  { label: t('dns.rtt'), value: `${props.result.rttMs} ms`, noCopy: true },
])
const columns = computed(() => [
  { key: 'name', title: t('dns.columns.name'), mono: true, width: 220 },
  { key: 'ttl', title: t('dns.columns.ttl'), width: 80, sortable: true },
  { key: 'type', title: t('dns.columns.type'), width: 90 },
  { key: 'data', title: t('dns.columns.data'), mono: true, width: 380 },
])
const sections = computed(() => ([
  ['answer', props.result.answer],
  ['authority', props.result.authority],
  ['additional', props.result.additional],
] as const).filter(([, rows]) => rows.length))
</script>

<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <NTag :type="rcodeType" :bordered="false" data-testid="rcode">
        {{ result.rcode }}
      </NTag>
      <span class="text-sm nt-muted">{{ t('dns.records', { count: result.answer.length }) }}</span>
    </div>
    <CKeyValueList :rows="summary" />
    <div v-for="[name, rows] in sections" :key="name" class="mt-4">
      <h3 class="mb-2 mt-0 text-sm font-semibold">
        {{ t(`dns.sections.${name}`) }}
      </h3>
      <ResultTable :columns="columns" :rows="rows" :label="t(`dns.sections.${name}`)" :filename="filename ?? 'dns.json'" />
    </div>
  </div>
</template>
