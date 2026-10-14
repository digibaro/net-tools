<script setup lang="ts">
import type { SshAudit } from './ssh-audit.service'
import { NAlert, NButton, NFormItem, NInputNumber, NTag } from 'naive-ui'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidHost } from '@/utils/host'
import { rating, severityOf } from './ssh-audit.service'

const { t } = useI18n()
const host = useQueryParam({ name: 'host', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 22 })
const validation = useValidation({ source: host, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const { data, error, loading, run } = useApi<SshAudit>('/v1/ssh/audit')
const submit = () => host.value && validation.value.isValid && run({ host: host.value.trim(), port: port.value })
onMounted(() => {
  if (host.value)
    void submit()
})
const ratingType = { good: 'success', fair: 'warning', weak: 'error' } as const
const sevType = { 'fail': 'error', 'warn': 'warning', '': 'default' } as const
const lists = ['kex', 'hostKey', 'ciphers', 'macs', 'compression'] as const
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="host" :label="t('tools.revocation-check.host')" placeholder="github.com" :validation="validation" monospace test-id="host" />
        <NFormItem :label="t('tools.tcp-ping.port')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.tcp-ping.port') }" />
        </NFormItem>
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!host || !validation.isValid" data-testid="run">
        {{ t('tools.ssh-audit.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.ssh-audit.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="ratingType[rating(data)]" class="mb-4" data-testid="result">
      <strong>{{ t(`tools.tls-scanner.rating.${rating(data)}`) }}</strong>
      <ul v-if="data.findings.length" class="mb-0 mt-1 pl-5">
        <li v-for="(f, i) in data.findings" :key="i">
          <span v-if="f.algorithm" class="font-mono">{{ f.algorithm }}: </span>{{ f.reason }}
        </li>
      </ul>
    </NAlert>
    <CCard :title="t('tools.ssh-audit.server')">
      <CKeyValueList :rows="[{ label: t('tools.ssh-audit.banner'), value: data.banner, mono: true }, { label: t('tools.revocation-check.host'), value: data.address, mono: true }]" />
    </CCard>
    <CCard :title="t('tools.ssh-audit.hostKeys')">
      <CKeyValueList v-if="data.hostKeys.length" :rows="data.hostKeys.map(k => ({ label: `${k.type} (${k.bits} bit)`, value: k.fingerprint, mono: true }))" />
      <p v-else class="m-0 text-sm">
        —
      </p>
    </CCard>
    <CCard :title="t('tools.ssh-audit.algorithms')">
      <div v-for="l in lists" :key="l" class="mb-3">
        <h3 class="mb-1 mt-0 text-sm font-semibold">
          {{ t(`tools.ssh-audit.lists.${l}`) }}
        </h3>
        <div class="flex flex-wrap gap-1">
          <NTag v-for="alg in data.algorithms[l]" :key="alg" size="small" :bordered="false" :type="sevType[severityOf(data, alg)]">
            <span class="font-mono">{{ alg }}</span>
          </NTag>
        </div>
      </div>
      <ResultActions :text="() => [data!.banner, ...data!.hostKeys.map(k => `${k.type} ${k.fingerprint}`), ...data!.findings.map(f => `${f.severity}: ${f.algorithm} ${f.reason}`)].join('\n')" :json="data" filename="ssh-audit.json" />
    </CCard>
  </template>
</template>
