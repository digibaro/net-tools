<script setup lang="ts">
import type { RelayResult } from './open-relay-test.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NFormItem, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiGet } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { buildQuery } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'
import { verdict } from './open-relay-test.service'

const { t } = useI18n()
const host = useQueryParam({ name: 'host', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 25 })
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const hostV = useValidation({ source: host, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const domainV = useValidation({ source: domain, rules: [{ validator: (s: string) => isValidHost(s) && s.includes('.'), message: t('tools.open-relay-test.invalidDomain') }] })
const result = shallowRef<RelayResult>()
const error = ref<ApiErrorType>()
const running = ref(false)
const canRun = computed(() => !!host.value && hostV.value.isValid && domainV.value.isValid)
async function run() {
  if (!canRun.value)
    return
  running.value = true
  error.value = undefined
  result.value = undefined
  try {
    result.value = await apiGet<RelayResult>(`/v1/email/relay${buildQuery({ host: host.value.trim(), port: port.value, domain: domain.value.trim() || undefined })}`)
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    running.value = false
  }
}
onMounted(() => {
  if (host.value)
    void run()
})
const alertType = { open: 'error', closed: 'success', inconclusive: 'warning' } as const
const rows = computed(() => (result.value?.cases ?? []).map(c => ({
  name: t(`tools.open-relay-test.cases.${c.name}`),
  from: c.mailFrom ? `<${c.mailFrom}>` : '<>',
  to: `<${c.rcptTo}>`,
  reply: c.rcptReply || c.mailReply,
  accepted: c.accepted,
  status: c.accepted ? t('tools.open-relay-test.relayed') : t('tools.open-relay-test.refused'),
  id: c.name,
})))
const c = (k: string) => t(`tools.open-relay-test.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="host" :label="t('tools.open-relay-test.host')" placeholder="mail.example.com" :validation="hostV" monospace test-id="host" />
        <CInputText v-model:value="domain" :label="t('tools.open-relay-test.domain')" placeholder="example.com" :validation="domainV" monospace test-id="domain" />
      </div>
      <NFormItem :label="t('tools.open-relay-test.port')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="port" name="relay-port">
          <NRadioButton :value="25" label="25" />
          <NRadioButton :value="587" label="587" />
          <NRadioButton :value="2525" label="2525" />
        </NRadioGroup>
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!canRun || running" data-testid="run">
        {{ t('tools.open-relay-test.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.open-relay-test.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <NAlert :type="alertType[verdict(result)]" class="mb-3">
      {{ t(`tools.open-relay-test.verdict.${verdict(result)}`, { error: result.error }) }}
    </NAlert>
    <p v-if="result.banner" class="mt-0 text-sm nt-muted font-mono">
      {{ result.address }} · {{ result.banner }}{{ result.startTls ? ' · STARTTLS' : '' }}
    </p>
    <ResultTable v-if="rows.length" :columns="[{ key: 'status', title: c('status'), width: 110 }, { key: 'name', title: c('case'), width: 180 }, { key: 'from', title: 'MAIL FROM', mono: true, width: 240 }, { key: 'to', title: 'RCPT TO', mono: true, width: 280 }, { key: 'reply', title: c('reply'), mono: true, width: 320 }]" :rows="rows" row-key="id" :actions="false" />
    <div class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.accepted ? 'RELAYED' : 'refused'}\t${r.from} -> ${r.to}\t${r.reply}`).join('\n')" :json="result" filename="relay-test.json" />
    </div>
  </CCard>
</template>
