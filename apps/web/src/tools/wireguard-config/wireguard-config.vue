<script setup lang="ts">
import type { Keys, Settings } from './wireguard-config.service'
import { NAlert, NButton, NFormItem, NInputNumber, NRadioButton, NRadioGroup, NSwitch } from 'naive-ui'
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { downloadText } from '@/utils/download'
import { qrSvg } from '../wifi-qr-generator/wifi-qr-generator.service'
import { buildConfigs, generateKeys, MAX_PEERS, validate } from './wireguard-config.service'

const { t } = useI18n()
// Network settings are shareable via the URL; keys are generated locally and never stored.
const endpoint = useQueryParam({ name: 'endpoint', defaultValue: 'vpn.example.com' })
const listenPort = useQueryParam({ name: 'port', defaultValue: 51820 })
const address4 = useQueryParam({ name: 'net4', defaultValue: '10.8.0.1/24' })
const address6 = useQueryParam({ name: 'net6', defaultValue: '' })
const dns = useQueryParam({ name: 'dns', defaultValue: '1.1.1.1, 1.0.0.1' })
const mtu = useQueryParam({ name: 'mtu', defaultValue: 0 })
const peers = useQueryParam({ name: 'peers', defaultValue: 2 })
const routing = useQueryParam({ name: 'routing', defaultValue: 'full' })
const allowedIps = useQueryParam({ name: 'allowed', defaultValue: '' })
const keepalive = useQueryParam({ name: 'keepalive', defaultValue: 25 })
const psk = useQueryParam({ name: 'psk', defaultValue: true })

const settings = computed<Settings>(() => ({
  endpoint: endpoint.value,
  listenPort: listenPort.value,
  address4: address4.value,
  address6: address6.value,
  dns: dns.value,
  mtu: mtu.value || null,
  peers: peers.value,
  routing: (['full', 'split', 'custom'].includes(routing.value) ? routing.value : 'full') as Settings['routing'],
  allowedIps: allowedIps.value,
  keepalive: keepalive.value,
  presharedKeys: psk.value,
}))
const error = computed(() => validate(settings.value))

const keys = shallowRef<Keys>(generateKeys(MAX_PEERS))
const regenerate = () => (keys.value = generateKeys(MAX_PEERS))
const configs = computed(() => (error.value ? undefined : buildConfigs(settings.value, keys.value)))

const qr = ref<Record<string, string>>({})
watch(configs, async (c) => {
  const out: Record<string, string> = {}
  for (const p of c?.peers ?? [])
    out[p.name] = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(await qrSvg(p.config))}`
  qr.value = out
}, { immediate: true })

const fieldError = (f: string) => (error.value === f ? { isValid: false, status: 'error' as const, message: t(`tools.wireguard-config.errors.${f}`) } : undefined)
const f = (key: string) => t(`tools.wireguard-config.fields.${key}`)
</script>

<template>
  <CCard>
    <div class="grid gap-x-4 sm:grid-cols-2">
      <CInputText v-model:value="endpoint" :label="f('endpoint')" placeholder="vpn.example.com" :validation="fieldError('endpoint')" monospace test-id="endpoint" />
      <NFormItem :label="f('port')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="listenPort" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': f('port') }" />
      </NFormItem>
      <CInputText v-model:value="address4" :label="f('address4')" placeholder="10.8.0.1/24" :validation="fieldError('address4') ?? fieldError('full')" monospace test-id="address4" />
      <CInputText v-model:value="address6" :label="f('address6')" placeholder="fd42:42::1/64" :validation="fieldError('address6')" monospace />
      <CInputText v-model:value="dns" :label="f('dns')" placeholder="1.1.1.1" :validation="fieldError('dns')" monospace />
      <div class="grid grid-cols-3 gap-x-4">
        <NFormItem :label="f('peers')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="peers" :show-button="false" :min="1" :max="MAX_PEERS" :input-props="{ 'aria-label': f('peers') }" data-testid="peers" />
        </NFormItem>
        <NFormItem :label="f('mtu')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="mtu" :min="0" :max="9000" :show-button="false" :placeholder="t('tools.wireguard-config.auto')" :input-props="{ 'aria-label': f('mtu') }" />
        </NFormItem>
        <NFormItem :label="f('keepalive')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="keepalive" :min="0" :max="3600" :show-button="false" :input-props="{ 'aria-label': f('keepalive') }" />
        </NFormItem>
      </div>
    </div>
    <NFormItem :label="f('routing')" :show-feedback="false" class="mb-3">
      <NRadioGroup v-model:value="routing" name="wg-routing">
        <NRadioButton value="full" :label="t('tools.wireguard-config.routing.full')" />
        <NRadioButton value="split" :label="t('tools.wireguard-config.routing.split')" />
        <NRadioButton value="custom" :label="t('tools.wireguard-config.routing.custom')" />
      </NRadioGroup>
    </NFormItem>
    <CInputText v-if="routing === 'custom'" v-model:value="allowedIps" :label="f('allowedIps')" placeholder="10.8.0.0/24, 192.168.1.0/24" :validation="fieldError('allowedIps')" monospace />
    <div class="flex flex-wrap items-center gap-x-6 gap-y-2">
      <NFormItem :label="f('psk')" :show-feedback="false" label-placement="left">
        <NSwitch v-model:value="psk" :aria-label="f('psk')" />
      </NFormItem>
      <NButton secondary data-testid="regenerate" @click="regenerate">
        {{ t('tools.wireguard-config.regenerate') }}
      </NButton>
    </div>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.wireguard-config.privacy') }}
    </p>
  </CCard>
  <NAlert v-if="error && !['endpoint', 'address4', 'address6', 'dns', 'allowedIps', 'full'].includes(error)" type="error" class="mb-4">
    {{ t(`tools.wireguard-config.errors.${error}`, { max: MAX_PEERS }) }}
  </NAlert>
  <template v-if="configs">
    <CCard :title="t('tools.wireguard-config.serverTitle')" data-testid="result">
      <pre class="m-0 max-h-96 overflow-auto text-xs font-mono" tabindex="0" data-testid="server-config">{{ configs.server }}</pre>
      <div class="mt-3 flex flex-wrap gap-2">
        <CopyButton :text="configs.server" :label="t('common.copyText')" />
        <NButton size="small" secondary @click="downloadText('wg0.conf', configs!.server, 'text/plain')">
          {{ t('tools.wireguard-config.download', { file: 'wg0.conf' }) }}
        </NButton>
      </div>
    </CCard>
    <CCard v-for="p in configs.peers" :key="p.name" :title="`${p.name} · ${p.address}`" data-testid="peer">
      <div class="flex flex-col gap-4 md:flex-row">
        <pre class="m-0 min-w-0 flex-1 overflow-auto text-xs font-mono" tabindex="0">{{ p.config }}</pre>
        <img v-if="qr[p.name]" :src="qr[p.name]" :alt="t('tools.wireguard-config.qrAlt', { name: p.name })" width="200" height="200" class="block self-start rounded bg-white">
      </div>
      <div class="mt-3 flex flex-wrap gap-2">
        <CopyButton :text="p.config" :label="t('common.copyText')" />
        <NButton size="small" secondary @click="downloadText(`${p.name}.conf`, p.config, 'text/plain')">
          {{ t('tools.wireguard-config.download', { file: `${p.name}.conf` }) }}
        </NButton>
      </div>
    </CCard>
  </template>
</template>
