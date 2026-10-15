<script setup lang="ts">
import type { LldpNeighbor } from './lldp-neighbors.service'
import { NButton, NFormItem, NInputNumber, NProgress } from 'naive-ui'
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { deviceName, isInterfaceName } from './lldp-neighbors.service'

const { t } = useI18n()
const iface = useQueryParam({ name: 'interface', defaultValue: '' })
const seconds = useQueryParam({ name: 'seconds', defaultValue: 35 })
const ifaceV = useValidation({ source: iface, rules: [{ validator: isInterfaceName, message: t('tools.lldp-neighbors.invalidInterface') }] })
const found = ref<LldpNeighbor[]>([])
const elapsed = ref(0)
let timer: ReturnType<typeof setInterval> | undefined
const { status, error, start, stop } = useSse('/v1/lan/lldp/stream', { neighbor: d => found.value.push(d as LldpNeighbor) })
const running = computed(() => status.value === 'running')
function run() {
  if (!ifaceV.value.isValid)
    return
  found.value = []
  elapsed.value = 0
  clearInterval(timer)
  timer = setInterval(() => {
    elapsed.value++
    if (elapsed.value >= seconds.value || status.value !== 'running')
      clearInterval(timer)
  }, 1000)
  start({ interface: iface.value.trim() || undefined, seconds: seconds.value })
}
onBeforeUnmount(() => clearInterval(timer))
const rows = computed(() => found.value.map(n => ({
  id: `${n.protocol}|${n.sourceMac}|${n.portId}`,
  device: deviceName(n),
  port: n.portDescription ? `${n.portId} (${n.portDescription})` : n.portId,
  protocol: n.protocol,
  vlan: n.vlan || '—',
  caps: (n.enabled.length ? n.enabled : n.capabilities).join(', ') || '—',
  mgmt: n.management.join(', ') || '—',
  platform: n.platform || n.systemDescription.split('\n')[0] || '—',
})))
const c = (k: string) => t(`tools.lldp-neighbors.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="iface" :label="t('tools.lldp-neighbors.interface')" placeholder="eth0" :validation="ifaceV" monospace test-id="interface" />
        <NFormItem :label="t('tools.lldp-neighbors.seconds')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="seconds" :min="5" :max="65" :show-button="false" :input-props="{ 'aria-label': t('tools.lldp-neighbors.seconds') }" />
        </NFormItem>
      </div>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="running || !ifaceV.isValid" data-testid="run">
          {{ t('tools.lldp-neighbors.run') }}
        </NButton>
        <NButton v-if="running" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.lldp-neighbors.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="running || status === 'done'" :title="t('tools.lldp-neighbors.found', { n: found.length })" data-testid="result">
    <NProgress v-if="running" type="line" :percentage="Math.min(100, Math.round(elapsed / seconds * 100))" :show-indicator="false" :aria-label="t('tools.lldp-neighbors.listening')" class="mb-3" />
    <p v-if="status === 'done' && !found.length" class="mt-0 nt-muted">
      {{ t('tools.lldp-neighbors.none', { seconds }) }}
    </p>
    <ResultTable v-if="rows.length" :columns="[{ key: 'device', title: c('device'), width: 200 }, { key: 'port', title: c('port'), mono: true, width: 220 }, { key: 'protocol', title: c('protocol'), width: 80 }, { key: 'vlan', title: 'VLAN', width: 70 }, { key: 'caps', title: c('capabilities'), width: 160 }, { key: 'mgmt', title: c('management'), mono: true, width: 160 }, { key: 'platform', title: c('platform'), width: 240 }]" :rows="rows" row-key="id" filename="lldp-neighbors.json" />
  </CCard>
</template>
