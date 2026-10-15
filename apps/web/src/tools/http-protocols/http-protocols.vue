<script setup lang="ts">
import type { HttpProtocols } from './http-protocols.service'
import { NAlert, NButton, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { fixed } from '@/utils/format'
import { best, isHttpsUrl } from './http-protocols.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: '' })
const validation = useValidation({ source: url, rules: [{ validator: isHttpsUrl, message: t('tools.http-protocols.httpsOnly') }] })
const { data, error, loading, run } = useApi<HttpProtocols>('/v1/http/protocols')
const submit = () => url.value && validation.value.isValid && run({ url: url.value.trim() })
onMounted(() => {
  if (url.value)
    void submit()
})
const yes = (b: boolean) => (b ? t('common.yes') : t('common.no'))
const rows = computed(() => {
  const d = data.value
  if (!d)
    return []
  const k = (key: string) => t(`tools.http-protocols.fields.${key}`)
  return [
    { label: k('http2'), value: `${yes(d.http2)} (ALPN: ${d.alpn || '—'}, ${d.tlsVersion})`, noCopy: true },
    { label: k('used'), value: d.httpVersion || '—', noCopy: true },
    { label: k('altSvc'), value: d.altSvc.map(a => `${a.protocol}="${a.host}:${a.port}" ma=${a.maxAge}`).join(', ') || '—', mono: true },
    { label: k('http3'), value: d.http3.ok ? t('tools.http-protocols.h3ok', { port: d.http3.port, ms: d.http3.handshakeMs, version: d.http3.quicVersion }) : d.http3.error, noCopy: true },
    ...(d.http3.request
      ? [
          { label: k('h3request'), value: d.http3.request.ok ? t('tools.http-protocols.h3req', { status: d.http3.request.status, ms: fixed(d.http3.request.ttfbMs, 1) }) : t('tools.http-protocols.h3reqFail', { error: d.http3.request.error }), noCopy: true },
          ...(d.http3.request.ok ? [{ label: k('resumption'), value: t('tools.http-protocols.resumed', { resumed: yes(d.http3.request.resumed), zeroRtt: yes(d.http3.request.zeroRtt), ms: fixed(d.http3.request.repeatTtfbMs, 1) }), noCopy: true }] : []),
        ]
      : []),
    ...(d.ttfbMs ? [{ label: k('ttfbTcp'), value: `${fixed(d.ttfbMs, 1)} ms (${d.httpVersion})`, noCopy: true }] : []),
    { label: k('address'), value: d.address, mono: true },
  ]
})
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="url" :label="t('http.url')" placeholder="https://example.com" :validation="validation" monospace test-id="url" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!url || !validation.isValid" data-testid="run">
        {{ t('tools.http-protocols.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="data.url" data-testid="result">
      <template #header-extra>
        <NTag :type="best(data) === 'HTTP/1.1' ? 'warning' : 'success'" :bordered="false">
          {{ best(data) }}
        </NTag>
      </template>
      <NAlert v-if="data.http3.ok && !data.http3.advertised" type="info" class="mb-3">
        {{ t('tools.http-protocols.notAdvertised') }}
      </NAlert>
      <NAlert v-else-if="data.http3.advertised && !data.http3.ok" type="warning" class="mb-3">
        {{ t('tools.http-protocols.advertisedBroken') }}
      </NAlert>
      <CKeyValueList :rows="rows" />
      <div class="mt-3">
        <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" filename="http-protocols.json" />
      </div>
    </CCard>
  </template>
</template>
