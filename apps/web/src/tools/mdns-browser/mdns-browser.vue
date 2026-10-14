<script setup lang="ts">
import type { MdnsService } from './mdns-browser.service'
import { NButton, NFormItem, NInputNumber } from 'naive-ui'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { describeType, instanceName, shortType } from './mdns-browser.service'

const { t } = useI18n()
const service = useQueryParam({ name: 'service', defaultValue: '' })
const seconds = useQueryParam({ name: 'seconds', defaultValue: 5 })
const found = ref<MdnsService[]>([])
const { status, error, start } = useSse('/v1/lan/mdns/stream', { service: d => found.value.push(d as MdnsService) })
const running = computed(() => status.value === 'running')
function run() {
  found.value = []
  start({ service: service.value.trim() || undefined, seconds: seconds.value })
}
const rows = computed(() => [...found.value].sort((a, b) => (a.type + a.instance).localeCompare(b.type + b.instance)).map(s => ({
  id: s.instance,
  name: instanceName(s),
  type: `${shortType(s.type)}${describeType(s.type) ? ` · ${describeType(s.type)}` : ''}`,
  host: `${s.host}:${s.port}`,
  addresses: s.addresses.join(', '),
  txt: s.txt.join(' '),
})))
const c = (k: string) => t(`tools.mdns-browser.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="service" :label="t('tools.mdns-browser.service')" placeholder="_http._tcp" monospace test-id="service" />
        <NFormItem :label="t('tools.mdns-browser.seconds')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="seconds" :min="1" :max="10" :show-button="false" :input-props="{ 'aria-label': t('tools.mdns-browser.seconds') }" />
        </NFormItem>
      </div>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="running" data-testid="run">
        {{ t('tools.mdns-browser.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.mdns-browser.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="found.length || status === 'done'" :title="t('tools.mdns-browser.found', { n: found.length })" data-testid="result">
    <ResultTable :columns="[{ key: 'name', title: c('name'), width: 200 }, { key: 'type', title: c('type'), width: 220 }, { key: 'host', title: c('host'), mono: true, width: 200 }, { key: 'addresses', title: c('addresses'), mono: true, width: 200 }, { key: 'txt', title: 'TXT', mono: true, width: 260 }]" :rows="rows" row-key="id" :actions="false" />
    <div v-if="status === 'done'" class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.name}\t${r.type}\t${r.host}\t${r.addresses}`).join('\n')" :json="found" filename="mdns.json" />
    </div>
  </CCard>
</template>
