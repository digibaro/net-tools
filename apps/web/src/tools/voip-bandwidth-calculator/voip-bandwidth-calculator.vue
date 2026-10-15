<script setup lang="ts">
import type { Codec, Link } from './voip-bandwidth-calculator.service'
import { NFormItem, NInputNumber, NSwitch } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { fixed } from '@/utils/format'
import { channelsFor, CODECS, erlangB, erlangs, LINKS, perCall } from './voip-bandwidth-calculator.service'

const { t } = useI18n()
const codec = useQueryParam({ name: 'codec', defaultValue: 'G.711' })
const ptime = useQueryParam({ name: 'ptime', defaultValue: 20 })
const link = useQueryParam({ name: 'link', defaultValue: 'ethernet' })
const ipv6 = useQueryParam({ name: 'ipv6', defaultValue: false })
const srtp = useQueryParam({ name: 'srtp', defaultValue: false })
const vlan = useQueryParam({ name: 'vlan', defaultValue: false })
const calls = useQueryParam({ name: 'calls', defaultValue: 10 })
const codecOptions = Object.keys(CODECS).map(c => ({ label: `${c} (${CODECS[c as Codec]} kbit/s)`, value: c }))
const linkOptions = computed(() => Object.keys(LINKS).map(l => ({ label: t(`tools.voip-bandwidth-calculator.links.${l}`), value: l })))
const r = computed(() => perCall({ codec: codec.value as Codec, ptimeMs: ptime.value, ipv6: ipv6.value, srtp: srtp.value, vlan: vlan.value, link: link.value as Link }))
const k = (key: string) => t(`tools.voip-bandwidth-calculator.fields.${key}`)
const bwRows = computed(() => [
  { label: k('payload'), value: `${r.value.payloadBytes} B`, noCopy: true },
  { label: k('packet'), value: `${r.value.packetBytes} B (${fixed(r.value.overheadPct, 0)} % ${t('tools.voip-bandwidth-calculator.overhead')})`, noCopy: true },
  { label: k('pps'), value: fixed(r.value.pps, 1), noCopy: true },
  { label: k('perCall'), value: `${fixed(r.value.kbps, 1)} kbit/s`, noCopy: true },
  { label: k('total'), value: `${fixed(r.value.kbps * calls.value / 1000, 2)} Mbit/s (${calls.value} × ${fixed(r.value.kbps, 1)} kbit/s)`, noCopy: true },
])
const busyCalls = useQueryParam({ name: 'bhca', defaultValue: 120 })
const avgMin = useQueryParam({ name: 'avg', defaultValue: 3 })
const blocking = useQueryParam({ name: 'gos', defaultValue: 1 })
const traffic = computed(() => erlangs(busyCalls.value, avgMin.value))
const channels = computed(() => channelsFor(traffic.value, blocking.value / 100))
const erlRows = computed(() => [
  { label: k('erlangs'), value: `${fixed(traffic.value, 2)} E`, noCopy: true },
  { label: k('channels'), value: String(channels.value), noCopy: true },
  { label: k('blocking'), value: `${fixed(erlangB(traffic.value, channels.value) * 100, 2)} %`, noCopy: true },
  { label: k('trunkBw'), value: `${fixed(r.value.kbps * channels.value / 1000, 2)} Mbit/s`, noCopy: true },
])
</script>

<template>
  <CCard :title="t('tools.voip-bandwidth-calculator.bandwidth')">
    <div class="grid gap-x-4 sm:grid-cols-2">
      <CSelect v-model:value="codec" :label="t('tools.voip-bandwidth-calculator.codec')" :options="codecOptions" test-id="codec" />
      <CSelect v-model:value="link" :label="t('tools.voip-bandwidth-calculator.link')" :options="linkOptions" test-id="link" />
      <NFormItem :label="t('tools.voip-bandwidth-calculator.ptime')" :show-feedback="false" class="mb-3">
        <NInputNumber :show-button="false" v-model:value="ptime" :min="10" :max="120" :step="10" :input-props="{ 'aria-label': t('tools.voip-bandwidth-calculator.ptime') }" />
      </NFormItem>
      <NFormItem :label="t('tools.voip-bandwidth-calculator.calls')" :show-feedback="false" class="mb-3">
        <NInputNumber :show-button="false" v-model:value="calls" :min="1" :max="100000" :input-props="{ 'aria-label': t('tools.voip-bandwidth-calculator.calls') }" />
      </NFormItem>
    </div>
    <div class="mb-3 flex flex-wrap gap-x-6 gap-y-2 text-sm">
      <label class="flex items-center gap-2"><NSwitch v-model:value="ipv6" size="small" aria-label="IPv6" /> IPv6</label>
      <label class="flex items-center gap-2"><NSwitch v-model:value="srtp" size="small" aria-label="SRTP" /> SRTP</label>
      <label class="flex items-center gap-2"><NSwitch v-model:value="vlan" size="small" aria-label="802.1Q VLAN" /> 802.1Q VLAN</label>
    </div>
    <CKeyValueList :rows="bwRows" data-testid="result" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.voip-bandwidth-calculator.note') }}
    </p>
  </CCard>
  <CCard :title="t('tools.voip-bandwidth-calculator.erlang')">
    <div class="grid gap-x-4 sm:grid-cols-3">
      <NFormItem :label="t('tools.voip-bandwidth-calculator.bhca')" :show-feedback="false" class="mb-3">
        <NInputNumber :show-button="false" v-model:value="busyCalls" :min="0" :input-props="{ 'aria-label': t('tools.voip-bandwidth-calculator.bhca') }" />
      </NFormItem>
      <NFormItem :label="t('tools.voip-bandwidth-calculator.avg')" :show-feedback="false" class="mb-3">
        <NInputNumber :show-button="false" v-model:value="avgMin" :min="0.1" :step="0.5" :input-props="{ 'aria-label': t('tools.voip-bandwidth-calculator.avg') }" />
      </NFormItem>
      <NFormItem :label="t('tools.voip-bandwidth-calculator.gos')" :show-feedback="false" class="mb-3">
        <NInputNumber :show-button="false" v-model:value="blocking" :min="0.01" :max="50" :step="0.5" :input-props="{ 'aria-label': t('tools.voip-bandwidth-calculator.gos') }" />
      </NFormItem>
    </div>
    <CKeyValueList :rows="erlRows" data-testid="erlang" />
  </CCard>
</template>
