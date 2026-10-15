<script setup lang="ts">
import type { OpenResolverResult } from './open-resolver-test.service'
import { NAlert, NButton } from 'naive-ui'
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
import { isServer, verdict } from './open-resolver-test.service'

const { t } = useI18n()
const server = useQueryParam({ name: 'server', defaultValue: '' })
const validation = useValidation({ source: server, rules: [{ validator: (s: string) => !s || isServer(s), message: t('tools.open-resolver-test.invalid') }] })
const { data, error, loading, run } = useApi<OpenResolverResult>('/v1/dns/open-resolver')
const submit = () => server.value && validation.value.isValid && run({ server: server.value.trim() })
onMounted(() => {
  if (server.value)
    void submit()
})
const v = computed(() => (data.value ? verdict(data.value) : undefined))
const alertType = { open: 'error', closed: 'success', silent: 'info' } as const
const k = (key: string) => t(`tools.open-resolver-test.fields.${key}`)
const yes = (b: boolean) => t(b ? 'common.yes' : 'common.no')
const rows = computed(() => {
  const d = data.value
  if (!d)
    return []
  const out = [{ label: k('server'), value: d.server, mono: true }]
  if (d.responds) {
    out.push(
      { label: k('rcode'), value: d.rcode, mono: true },
      { label: k('ra'), value: yes(d.recursionAvailable), mono: false },
      { label: k('answers'), value: String(d.answers), mono: false },
      { label: k('rtt'), value: `${fixed(d.rttMs, 1)} ms`, mono: false },
    )
  }
  if (d.amplification) {
    const a = d.amplification
    out.push({ label: k('amplification'), value: t('tools.open-resolver-test.amplificationValue', { factor: fixed(a.factor, 1), req: a.requestBytes, resp: a.responseBytes, query: a.query }), mono: false })
  }
  return out
})
const asText = () => [v.value ? t(`tools.open-resolver-test.verdict.${v.value}`) : '', ...rows.value.map(r => `${r.label}: ${r.value}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="server" :label="t('tools.open-resolver-test.server')" placeholder="203.0.113.53" :validation="validation" monospace test-id="server" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!server || !validation.isValid" data-testid="run">
        {{ t('tools.open-resolver-test.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.open-resolver-test.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data && v" :title="t('common.result')" data-testid="result">
    <NAlert :type="alertType[v]" class="mb-3" data-testid="verdict">
      {{ t(`tools.open-resolver-test.verdict.${v}`) }}
    </NAlert>
    <CKeyValueList :rows="rows" />
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="open-resolver.json" />
    </div>
  </CCard>
</template>
