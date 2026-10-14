<script setup lang="ts">
import type { Revocation } from './revocation-check.service'
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
import { isValidHost } from '@/utils/host'
import { crlOnly, STATUS_TYPE } from './revocation-check.service'

const { t } = useI18n()
const host = useQueryParam({ name: 'host', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 443 })
const validation = useValidation({ source: host, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const { data, error, loading, run } = useApi<Revocation>('/v1/tls/revocation')
const submit = () => host.value && validation.value.isValid && run({ host: host.value.trim(), port: port.value })
onMounted(() => {
  if (host.value)
    void submit()
})
const status = (s: string) => t(`tools.revocation-check.status.${s}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="host" :label="t('tools.revocation-check.host')" placeholder="example.com" :validation="validation" monospace test-id="host" />
        <NFormItem :label="t('tools.tcp-ping.port')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.tcp-ping.port') }" />
        </NFormItem>
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!host || !validation.isValid" data-testid="run">
        {{ t('tools.revocation-check.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="STATUS_TYPE[data.status]" class="mb-4" data-testid="result">
      {{ t(`tools.revocation-check.verdict.${data.status}`, { cn: data.subject, issuer: data.issuer }) }}
    </NAlert>
    <CCard :title="t('tools.revocation-check.ocsp')">
      <p v-if="!data.ocsp.length" class="m-0 text-sm">
        {{ t('tools.revocation-check.noOcsp') }}
      </p>
      <div v-for="o in data.ocsp" :key="o.source" class="mb-2">
        <div class="flex flex-wrap items-center gap-2">
          <NTag size="small" :bordered="false" :type="STATUS_TYPE[o.status as keyof typeof STATUS_TYPE] ?? 'default'">
            {{ status(o.status) }}
          </NTag>
          <span class="break-all text-sm font-mono">{{ o.source === 'stapled' ? t('tools.revocation-check.stapled') : o.source }}</span>
        </div>
        <p class="m-0 mt-1 text-xs nt-muted">
          <template v-if="o.error">
            {{ o.error }}
          </template>
          <template v-else>
            {{ t('tools.revocation-check.updated', { this: o.thisUpdate, next: o.nextUpdate || '—' }) }}<template v-if="o.revokedAt">
              · {{ t('tools.revocation-check.revokedAt', { at: o.revokedAt, reason: o.reason }) }}
            </template>
          </template>
        </p>
      </div>
    </CCard>
    <CCard :title="t('tools.revocation-check.crl')">
      <p v-if="!data.crl.length" class="m-0 text-sm">
        {{ t('tools.revocation-check.noCrl') }}
      </p>
      <div v-for="c in data.crl" :key="c.url" class="mb-2">
        <div class="flex flex-wrap items-center gap-2">
          <NTag size="small" :bordered="false" :type="STATUS_TYPE[c.status as keyof typeof STATUS_TYPE] ?? 'default'">
            {{ status(c.status) }}
          </NTag>
          <span class="break-all text-sm font-mono">{{ c.url }}</span>
        </div>
        <p class="m-0 mt-1 text-xs nt-muted">
          <template v-if="c.status !== 'unknown'">
            {{ t('tools.revocation-check.crlInfo', { n: c.entries, this: c.thisUpdate, next: c.nextUpdate }) }}<template v-if="c.revokedAt">
              · {{ t('tools.revocation-check.revokedAt', { at: c.revokedAt, reason: c.reason }) }}
            </template>
          </template>
          <template v-if="c.error">
            {{ c.error }}
          </template>
        </p>
      </div>
      <p v-if="crlOnly(data)" class="mb-0 text-xs nt-muted">
        {{ t('tools.revocation-check.crlOnly') }}
      </p>
    </CCard>
    <ResultActions :text="() => `${data!.host}: ${data!.status} (serial ${data!.serial})`" :json="data" filename="revocation.json" />
  </template>
</template>
