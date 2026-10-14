<script setup lang="ts">
import type { WifiSecurity } from './wifi-qr-generator.service'
import { Download } from '@vicons/tabler'
import { watchDebounced } from '@vueuse/core'
import { NButton, NCheckbox, NIcon } from 'naive-ui'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { qrPngDataUrl, qrSvg, SECURITIES, validate, wifiUri } from './wifi-qr-generator.service'

// The password is deliberately kept out of the URL; only SSID, security and hidden are shareable.
const { t } = useI18n()
const ssid = useQueryParam({ name: 'ssid', defaultValue: '' })
const securityParam = useQueryParam({ name: 'security', defaultValue: 'WPA' })
const hidden = useQueryParam({ name: 'hidden', defaultValue: false })
const password = ref('')
const security = computed<WifiSecurity>({
  get: () => (SECURITIES.includes(securityParam.value as WifiSecurity) ? securityParam.value as WifiSecurity : 'WPA'),
  set: (v) => {
    securityParam.value = v
  },
})
const options = computed(() => SECURITIES.map(s => ({ label: t(`tools.wifi-qr-generator.security.${s}`), value: s })))
const config = computed(() => ({ ssid: ssid.value, password: password.value, security: security.value, hidden: hidden.value }))
const error = computed(() => validate(config.value))
const uri = computed(() => (error.value ? '' : wifiUri(config.value)))
const svg = ref('')

watchDebounced(uri, async (u) => {
  svg.value = u ? await qrSvg(u) : ''
}, { debounce: 150, immediate: true })

const svgUrl = computed(() => (svg.value ? `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg.value)}` : ''))
async function downloadPng() {
  const a = document.createElement('a')
  a.href = await qrPngDataUrl(uri.value)
  a.download = `wifi-${ssid.value.replace(/[^\w-]+/g, '_') || 'network'}.png`
  a.click()
}
const fieldError = (f: 'ssid' | 'password') => (ssid.value && error.value === f ? { isValid: false, status: 'error' as const, message: t(`tools.wifi-qr-generator.invalid.${f}`) } : undefined)
</script>

<template>
  <div class="grid gap-x-4 md:grid-cols-[1fr_auto]">
    <CCard>
      <CInputText v-model:value="ssid" :label="t('tools.wifi-qr-generator.ssid')" :validation="fieldError('ssid')" test-id="ssid" />
      <CSelect v-model:value="security" :label="t('tools.wifi-qr-generator.securityLabel')" :options="options" />
      <CInputText v-if="security !== 'nopass'" v-model:value="password" :label="t('tools.wifi-qr-generator.password')" :validation="fieldError('password')" test-id="password" />
      <NCheckbox v-model:checked="hidden">
        {{ t('tools.wifi-qr-generator.hidden') }}
      </NCheckbox>
      <p class="mb-0 text-sm nt-muted">
        {{ t('tools.wifi-qr-generator.privacy') }}
      </p>
    </CCard>
    <CCard v-if="svgUrl" :title="t('tools.wifi-qr-generator.qr')" data-testid="result">
      <img :src="svgUrl" :alt="t('tools.wifi-qr-generator.qrAlt', { ssid })" width="240" height="240" class="block rounded" data-testid="qr">
      <div class="mt-3 flex flex-wrap gap-2">
        <NButton size="small" secondary @click="downloadPng">
          <template #icon>
            <NIcon aria-hidden="true" :component="Download" />
          </template>
          {{ t('tools.wifi-qr-generator.downloadPng') }}
        </NButton>
        <CopyButton :text="uri" :label="t('tools.wifi-qr-generator.copyUri')" />
      </div>
    </CCard>
  </div>
</template>
