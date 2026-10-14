<script setup lang="ts">
import type { DnsEncrypted, EncryptedTransport } from './encrypted-dns-tester.service'
import { NAlert, NButton, NFormItem, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import DnsAnswer from '@/components/DnsAnswer.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { daysUntil, isValidEndpoint } from './encrypted-dns-tester.service'

const { t } = useI18n()
const endpoint = useQueryParam({ name: 'endpoint', defaultValue: 'cloudflare' })
const transport = useQueryParam({ name: 'transport', defaultValue: 'dot' })
const name = useQueryParam({ name: 'name', defaultValue: 'example.com' })
const valid = computed(() => isValidEndpoint(endpoint.value, transport.value as EncryptedTransport))
const { data, error, loading, run } = useApi<DnsEncrypted>('/v1/dns/encrypted')
const submit = () => valid.value && run({ endpoint: endpoint.value, transport: transport.value, name: name.value, count: 3 })
onMounted(() => {
  if (endpoint.value && endpoint.value !== 'cloudflare')
    void submit()
})

const tlsRows = computed(() => {
  const x = data.value?.tls
  if (!x)
    return []
  const k = (key: string) => t(`tools.encrypted-dns-tester.fields.${key}`)
  return [
    { label: k('address'), value: data.value!.address, mono: true },
    { label: k('protocol'), value: `${x.version} · ${x.cipher}${x.alpn ? ` · ALPN ${x.alpn}` : ''}`, noCopy: true },
    { label: k('handshake'), value: `${x.handshakeMs} ms`, noCopy: true },
    { label: k('subject'), value: x.subject, mono: true },
    { label: k('issuer'), value: x.issuer, mono: true },
    { label: k('names'), value: x.dnsNames.join(', '), mono: true },
    { label: k('expires'), value: x.notAfter ? `${x.notAfter} (${t('tools.encrypted-dns-tester.days', { days: daysUntil(x.notAfter) })})` : '—', noCopy: true },
    { label: k('latency'), value: data.value!.queries.map(q => (q.error ? '✕' : `${q.rttMs} ms`)).join(', ') + (data.value!.avgMs ? ` (avg ${data.value!.avgMs} ms)` : ''), noCopy: true },
  ]
})
const answer = computed(() => data.value && ({
  name: name.value,
  type: 'A',
  resolver: data.value.endpoint,
  transport: data.value.transport,
  rcode: data.value.queries.find(q => !q.error)?.rcode ?? '—',
  flags: { aa: false, tc: false, rd: true, ra: true, ad: false, cd: false },
  answer: data.value.answer,
  authority: [],
  additional: [],
  rttMs: data.value.avgMs,
}))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <NFormItem :label="t('tools.dns-lookup.transport')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="transport" name="transport">
          <NRadioButton value="dot" :label="t('tools.dns-lookup.transports.dot')" />
          <NRadioButton value="doh" :label="t('tools.dns-lookup.transports.doh')" />
        </NRadioGroup>
      </NFormItem>
      <div class="grid gap-x-4 sm:grid-cols-[1fr_14rem]">
        <CInputText
          v-model:value="endpoint" :label="t('tools.encrypted-dns-tester.endpoint')"
          :placeholder="transport === 'doh' ? 'https://dns.example/dns-query' : 'dns.example:853'"
          :validation="endpoint && !valid ? { isValid: false, status: 'error', message: t('tools.encrypted-dns-tester.invalid') } : undefined"
          monospace test-id="endpoint"
        />
        <CInputText v-model:value="name" :label="t('tools.encrypted-dns-tester.queryName')" monospace />
      </div>
      <p class="mt-0 text-xs nt-muted">
        {{ t('tools.encrypted-dns-tester.hint') }}
      </p>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!valid" data-testid="run">
        {{ t('tools.encrypted-dns-tester.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.tls.verified ? 'success' : 'error'" class="mb-4" data-testid="verdict">
      {{ data.tls.verified ? t('tools.encrypted-dns-tester.verified') : t('tools.encrypted-dns-tester.notVerified', { error: data.tls.verifyError }) }}
    </NAlert>
    <CCard :title="t('tools.encrypted-dns-tester.tls')" data-testid="result">
      <CKeyValueList :rows="tlsRows" />
    </CCard>
    <CCard v-if="answer" :title="t('dns.sections.answer')">
      <DnsAnswer :result="answer" filename="encrypted-dns.json" />
    </CCard>
  </template>
</template>
