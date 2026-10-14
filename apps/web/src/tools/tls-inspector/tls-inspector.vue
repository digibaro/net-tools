<script setup lang="ts">
import type { TlsSession } from './tls-inspector.service'
import { NAlert, NButton, NFormItem, NInputNumber, NTag } from 'naive-ui'
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { isValidHost } from '@/utils/host'
import { DEFAULT_PORTS, expiryType, isLegacyVersion, STARTTLS } from './tls-inspector.service'

/** Also used by the STARTTLS check (TLS-09): mode "starttls" starts with SMTP and hides direct TLS. */
const props = withDefaults(defineProps<{ mode?: 'tls' | 'starttls' }>(), { mode: 'tls' })
const { t } = useI18n()
const host = useQueryParam({ name: 'host', defaultValue: '' })
const starttls = useQueryParam({ name: 'starttls', defaultValue: props.mode === 'starttls' ? 'smtp' : '' })
const port = useQueryParam({ name: 'port', defaultValue: DEFAULT_PORTS[starttls.value] ?? 443 })
const sni = useQueryParam({ name: 'sni', defaultValue: '' })
watch(starttls, s => (port.value = DEFAULT_PORTS[s] ?? 443))
const hostValidation = useValidation({ source: () => host.value.replace(/:\d+$/, ''), rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const starttlsOptions = computed(() => STARTTLS.filter(s => props.mode === 'tls' || s).map(s => ({ label: s ? s.toUpperCase() : t('tools.tls-inspector.directTls'), value: s })))
const { data, error, loading, run } = useApi<TlsSession>('/v1/tls/inspect')
const submit = () => host.value && hostValidation.value.isValid && run({ host: host.value, port: port.value, sni: sni.value || undefined, starttls: starttls.value || undefined })
onMounted(() => {
  if (host.value)
    void submit()
})
const session = computed(() => data.value
  ? [
      { label: t('tools.tls-inspector.fields.connected'), value: data.value.connectedTo, mono: true },
      { label: t('tools.tls-inspector.fields.protocol'), value: `${data.value.version} · ${data.value.cipher}`, noCopy: true },
      { label: 'ALPN', value: data.value.alpn || '—', noCopy: true },
      { label: 'SNI', value: data.value.serverName, mono: true },
      { label: t('tools.tls-inspector.fields.ocsp'), value: t(data.value.ocspStapled ? 'common.yes' : 'common.no'), noCopy: true },
      { label: t('tools.tls-inspector.fields.handshake'), value: `${data.value.handshakeMs} ms`, noCopy: true },
    ]
  : [])
function certRows(c: TlsSession['chain'][number]) {
  const k = (key: string) => t(`tools.tls-inspector.cert.${key}`)
  return [
    { label: k('subject'), value: c.subject, mono: true },
    { label: k('issuer'), value: c.issuer, mono: true },
    ...(c.dnsNames.length || c.ipAddresses.length ? [{ label: k('san'), value: [...c.dnsNames, ...c.ipAddresses].join(', '), mono: true }] : []),
    { label: k('validity'), value: `${c.notBefore.slice(0, 10)} → ${c.notAfter.slice(0, 10)}`, noCopy: true },
    { label: k('key'), value: `${c.keyAlgorithm} · ${c.signatureAlgorithm}`, noCopy: true },
    { label: k('serial'), value: c.serial, mono: true },
    { label: 'SHA-256', value: c.sha256, mono: true },
  ]
}
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_7rem_10rem]">
        <CInputText v-model:value="host" :label="t('tools.tls-inspector.host')" placeholder="example.com" :validation="hostValidation" monospace test-id="host" />
        <NFormItem :label="t('tools.tcp-ping.port')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.tcp-ping.port') }" />
        </NFormItem>
        <CSelect v-model:value="starttls" :label="t('tools.tls-inspector.starttls')" :options="starttlsOptions" />
      </div>
      <CInputText v-model:value="sni" :label="t('tools.tls-inspector.sni')" :placeholder="t('tools.tls-inspector.sniPlaceholder')" monospace />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!host || !hostValidation.isValid" data-testid="run">
        {{ t('tools.tls-inspector.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.verified ? 'success' : 'error'" class="mb-4" data-testid="verdict">
      {{ data.verified ? t('tools.tls-inspector.valid', { days: data.chain[0]?.daysLeft ?? 0 }) : data.verifyError }}
      <template v-if="isLegacyVersion(data.version)">
        {{ t('tools.tls-inspector.legacy', { version: data.version }) }}
      </template>
    </NAlert>
    <CCard :title="t('tools.tls-inspector.session')" data-testid="result">
      <CKeyValueList :rows="session" />
    </CCard>
    <CCard v-for="(c, i) in data.chain" :key="c.sha256" :title="i === 0 ? t('tools.tls-inspector.leaf') : t('tools.tls-inspector.intermediate', { n: i })">
      <template #header-extra>
        <NTag :type="expiryType(c.daysLeft)" size="small" :bordered="false">
          {{ c.daysLeft < 0 ? t('tools.tls-inspector.expired') : t('tools.tls-inspector.daysLeft', { days: c.daysLeft }) }}
        </NTag>
      </template>
      <CKeyValueList :rows="certRows(c)" />
    </CCard>
    <ResultActions :text="() => JSON.stringify(data, null, 2)" :json="data" filename="tls.json" />
  </template>
</template>
