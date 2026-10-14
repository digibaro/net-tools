<script setup lang="ts">
import type { SsdpDevice } from './ssdp-discovery.service'
import { NButton, NFormItem, NInputNumber } from 'naive-ui'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import CCard from '@/ui/c-card.vue'
import { label, shortUrn } from './ssdp-discovery.service'

const { t } = useI18n()
const seconds = useQueryParam({ name: 'seconds', defaultValue: 4 })
const devices = ref<SsdpDevice[]>([])
const { status, error, start } = useSse('/v1/lan/ssdp/stream', { device: d => devices.value.push(d as SsdpDevice) })
const running = computed(() => status.value === 'running')
function run() {
  devices.value = []
  start({ seconds: seconds.value })
}
const rows = computed(() => devices.value.map(d => ({
  id: d.location,
  name: label(d),
  address: d.address,
  vendor: [d.manufacturer, d.model].filter(Boolean).join(' · ') || '—',
  types: [...new Set(d.types.map(shortUrn))].join(', '),
  server: d.server || '—',
  note: d.error ? t('tools.ssdp-discovery.descError') : '',
})))
const c = (k: string) => t(`tools.ssdp-discovery.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <NFormItem :label="t('tools.ssdp-discovery.seconds')" :show-feedback="false" class="mb-3 max-w-40">
        <NInputNumber v-model:value="seconds" :min="1" :max="10" :show-button="false" :input-props="{ 'aria-label': t('tools.ssdp-discovery.seconds') }" />
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="running" data-testid="run">
        {{ t('tools.ssdp-discovery.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.ssdp-discovery.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="devices.length || status === 'done'" :title="t('tools.ssdp-discovery.found', { n: devices.length })" data-testid="result">
    <ResultTable :columns="[{ key: 'name', title: c('name'), width: 200 }, { key: 'address', title: c('address'), mono: true, width: 140 }, { key: 'vendor', title: c('vendor'), width: 200 }, { key: 'types', title: c('types'), width: 260 }, { key: 'server', title: c('server'), mono: true, width: 220 }, { key: 'note', title: c('note'), width: 180 }]" :rows="rows" row-key="id" :actions="false" />
    <div v-if="status === 'done'" class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.address}\t${r.name}\t${r.vendor}\t${r.types}`).join('\n')" :json="devices" filename="ssdp.json" />
    </div>
  </CCard>
</template>
