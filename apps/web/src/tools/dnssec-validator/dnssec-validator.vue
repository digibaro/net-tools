<script setup lang="ts">
import type { DnssecResult } from './dnssec-validator.service'
import { NAlert, NButton, NTag } from 'naive-ui'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { keyLine, STATUS_TYPE, TYPES } from './dnssec-validator.service'

const { t } = useI18n()
const name = useQueryParam({ name: 'name', defaultValue: '' })
const type = useQueryParam({ name: 'type', defaultValue: 'A' })
const validation = useValidation({ source: name, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<DnssecResult>('/v1/dns/dnssec')
const submit = () => name.value && validation.value.isValid && run({ name: name.value.trim(), type: type.value })
onMounted(() => {
  if (name.value)
    void submit()
})
const typeOptions = TYPES.map(v => ({ label: v, value: v }))
const asText = () => data.value ? [`${data.value.name} ${data.value.type}: ${data.value.status}`, ...data.value.steps.map(s => `${s.zone}\t${s.status}\t${s.message}`), `answer\t${data.value.answer.status}\t${data.value.answer.message}`].join('\n') : ''
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_9rem]">
        <CInputText v-model:value="name" :label="t('tools.dns-lookup.name')" placeholder="example.com" :validation="validation" monospace test-id="name" />
        <CSelect v-model:value="type" :label="t('tools.dns-lookup.type')" :options="typeOptions" />
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!name || !validation.isValid" data-testid="run">
        {{ t('tools.dnssec-validator.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="STATUS_TYPE[data.status] === 'default' ? 'info' : STATUS_TYPE[data.status]" class="mb-4" data-testid="result">
      <strong>{{ t(`tools.dnssec-validator.status.${data.status}`) }}</strong> — {{ data.message }}
      <span v-if="data.resolverAd" class="nt-muted"> · {{ t('tools.dnssec-validator.ad') }}</span>
    </NAlert>
    <CCard :title="t('tools.dnssec-validator.chain')">
      <ol class="m-0 list-none p-0">
        <li v-for="s in data.steps" :key="s.zone" class="mb-3 border-l-4 nt-border border-solid pl-3" data-testid="step">
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-semibold font-mono">{{ s.zone }}</span>
            <NTag size="small" :bordered="false" :type="STATUS_TYPE[s.status]">
              {{ t(`tools.dnssec-validator.status.${s.status}`) }}
            </NTag>
          </div>
          <p class="m-0 mt-1 text-sm">
            {{ s.message }}
          </p>
          <p v-if="s.ds.length" class="m-0 mt-1 text-xs nt-muted font-mono">
            DS: {{ s.ds.map(d => `${d.tag} ${d.algorithm}/${d.digestType}${d.matched ? ' ✓' : ' ✗'}`).join(' · ') }}
          </p>
          <p v-if="s.keys.length" class="m-0 text-xs nt-muted font-mono">
            DNSKEY: {{ s.keys.map(keyLine).join(' · ') }}
          </p>
          <p v-for="(sig, i) in s.signatures.filter(x => !x.valid)" :key="i" class="m-0 text-xs">
            RRSIG {{ sig.covers }} ({{ sig.keyTag }}): {{ sig.error }}
          </p>
        </li>
        <li class="border-l-4 nt-border border-solid pl-3">
          <div class="flex flex-wrap items-center gap-2">
            <span class="font-semibold">{{ t('tools.dnssec-validator.answer', { name: data.name, type: data.type }) }}</span>
            <NTag size="small" :bordered="false" :type="STATUS_TYPE[data.answer.status as keyof typeof STATUS_TYPE] ?? 'default'">
              {{ data.answer.rcode }}
            </NTag>
          </div>
          <p class="m-0 mt-1 text-sm">
            {{ data.answer.message }}
          </p>
          <pre v-if="data.answer.records.length" class="m-0 mt-1 overflow-auto text-xs font-mono">{{ data.answer.records.map(r => `${r.name} ${r.ttl} ${r.type} ${r.data}`).join('\n') }}</pre>
          <p v-for="(sig, i) in data.answer.signatures.filter(x => !x.valid)" :key="i" class="m-0 text-xs">
            RRSIG {{ sig.covers }} ({{ sig.keyTag }}): {{ sig.error }}
          </p>
        </li>
      </ol>
      <div class="mt-3">
        <ResultActions :text="asText" :json="data" filename="dnssec.json" />
      </div>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.dnssec-validator.note') }}
      </p>
    </CCard>
  </template>
</template>
