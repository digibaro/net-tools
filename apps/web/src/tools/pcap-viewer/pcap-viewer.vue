<script setup lang="ts">
import type { Capture, CaptureError } from './pcap-viewer.service'
import { Upload } from '@vicons/tabler'
import { NAlert, NButton, NFormItem, NIcon, NInput, NInputNumber, NTabPane, NTabs } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { fixed } from '@/utils/format'
import { formatCount } from '@/utils/ip'
import { MAX_PACKETS, readCapture } from './pcap-viewer.service'

const MAX_ROWS = 5000
const { t } = useI18n()
const fileInput = ref<HTMLInputElement>()
const fileName = ref('')
const capture = shallowRef<Capture>()
const failure = ref<CaptureError>()
const filter = ref('')
const selected = ref<number | null>(1)
async function onFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file)
    return
  fileName.value = file.name
  const r = readCapture(await file.arrayBuffer())
  failure.value = typeof r === 'string' ? r : undefined
  capture.value = typeof r === 'string' ? undefined : r
  selected.value = 1
}
const matching = computed(() => {
  const f = filter.value.trim().toLowerCase()
  const all = capture.value?.packets ?? []
  return f ? all.filter(p => `${p.src} ${p.dst} ${p.protocol} ${p.info}`.toLowerCase().includes(f)) : all
})
const rows = computed(() => matching.value.slice(0, MAX_ROWS).map(p => ({ no: p.no, time: fixed(p.time, 6), src: p.src || '—', dst: p.dst || '—', protocol: p.protocol, len: p.len, info: p.info })))
const packet = computed(() => (selected.value ? capture.value?.packets[selected.value - 1] : undefined))
const k = (key: string) => t(`tools.pcap-viewer.columns.${key}`)
const summary = computed(() => capture.value
  ? [
      { label: k('file'), value: `${fileName.value} (${capture.value.format}, link type ${capture.value.linktypes.join(', ')})`, noCopy: true },
      { label: k('packets'), value: formatCount(BigInt(capture.value.packets.length)) + (capture.value.truncated ? ` (${t('tools.pcap-viewer.limited', { max: MAX_PACKETS })})` : ''), noCopy: true },
      { label: k('bytes'), value: formatCount(BigInt(capture.value.bytes)), noCopy: true },
      { label: k('duration'), value: `${fixed(capture.value.duration, 3)} s`, noCopy: true },
    ]
  : [])
</script>

<template>
  <CCard>
    <div class="flex flex-wrap items-center gap-2">
      <input ref="fileInput" type="file" accept=".pcap,.pcapng,.cap,application/vnd.tcpdump.pcap" class="hidden" :aria-label="t('tools.pcap-viewer.upload')" data-testid="file" @change="onFile">
      <NButton @click="fileInput?.click()">
        <template #icon>
          <NIcon aria-hidden="true" :component="Upload" />
        </template>
        {{ t('tools.pcap-viewer.upload') }}
      </NButton>
      <span class="text-sm font-mono">{{ fileName }}</span>
    </div>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.pcap-viewer.note') }}
    </p>
  </CCard>
  <NAlert v-if="failure" type="error" class="mb-4" data-testid="failure">
    {{ t(`tools.pcap-viewer.errors.${failure}`) }}
  </NAlert>
  <template v-if="capture">
    <CCard :title="t('tools.pcap-viewer.summary')" data-testid="result">
      <CKeyValueList :rows="summary" />
    </CCard>
    <CCard>
      <NTabs type="line" animated>
        <NTabPane name="packets" :tab="t('tools.pcap-viewer.tabs.packets')">
          <NInput v-model:value="filter" :placeholder="t('tools.pcap-viewer.filter')" clearable class="mb-3" :input-props="{ 'aria-label': t('tools.pcap-viewer.filter') }" data-testid="filter" />
          <p v-if="matching.length > MAX_ROWS" class="mt-0 text-xs nt-muted">
            {{ t('tools.pcap-viewer.firstRows', { n: MAX_ROWS, total: matching.length }) }}
          </p>
          <ResultTable :columns="[{ key: 'no', title: '#', width: 70 }, { key: 'time', title: k('time'), width: 110 }, { key: 'src', title: k('src'), mono: true, width: 180 }, { key: 'dst', title: k('dst'), mono: true, width: 180 }, { key: 'protocol', title: k('protocol'), width: 90 }, { key: 'len', title: k('len'), width: 70 }, { key: 'info', title: k('info'), width: 380 }]" :rows="rows" row-key="no" filename="packets.json" :max-height="480" />
        </NTabPane>
        <NTabPane name="protocols" :tab="t('tools.pcap-viewer.tabs.protocols')">
          <ResultTable :columns="[{ key: 'name', title: k('protocol'), width: 160 }, { key: 'packets', title: k('packets'), width: 100 }]" :rows="capture.protocols" row-key="name" filename="protocols.json" />
        </NTabPane>
        <NTabPane name="conversations" :tab="t('tools.pcap-viewer.tabs.conversations')">
          <ResultTable :columns="[{ key: 'a', title: 'A', mono: true, width: 220 }, { key: 'b', title: 'B', mono: true, width: 220 }, { key: 'packets', title: k('packets'), width: 100 }, { key: 'bytes', title: k('bytes'), width: 110 }]" :rows="capture.conversations.map(c => ({ ...c, id: `${c.a}|${c.b}` }))" row-key="id" filename="conversations.json" />
        </NTabPane>
        <NTabPane name="dns" :tab="t('tools.pcap-viewer.tabs.dns')">
          <ResultTable v-if="capture.dns.length" :columns="[{ key: 'query', title: k('query'), mono: true, width: 360 }, { key: 'packets', title: k('packets'), width: 100 }]" :rows="capture.dns" row-key="query" filename="dns.json" />
          <p v-else class="text-sm nt-muted">
            {{ t('tools.pcap-viewer.noDns') }}
          </p>
        </NTabPane>
      </NTabs>
    </CCard>
    <CCard :title="t('tools.pcap-viewer.details')">
      <NFormItem :label="t('tools.pcap-viewer.packetNo')" :show-feedback="false" class="mb-3 w-40">
        <NInputNumber v-model:value="selected" :min="1" :max="capture.packets.length" :input-props="{ 'aria-label': t('tools.pcap-viewer.packetNo') }" data-testid="packet-no" />
      </NFormItem>
      <template v-if="packet">
        <div v-for="(l, i) in packet.decoded.layers" :key="i" class="mb-3" data-testid="layer">
          <div class="mb-1 font-semibold">
            {{ l.name }} — {{ l.summary }}
          </div>
          <CKeyValueList v-if="l.fields.length" :rows="l.fields.map(f => ({ label: f.name, value: f.value, mono: true }))" />
        </div>
        <NAlert v-if="packet.decoded.error" type="warning">
          {{ packet.decoded.error }}
        </NAlert>
      </template>
    </CCard>
  </template>
</template>
