<script setup lang="ts">
import type { DnsCaa } from './caa-checker.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { verdict, wildcardCas } from './caa-checker.service'

const { t } = useI18n()
const name = useQueryParam({ name: 'name', defaultValue: '' })
const validation = useValidation({ source: name, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<DnsCaa>('/v1/dns/caa')
const submit = () => name.value && validation.value.isValid && run({ name: name.value })
onMounted(() => {
  if (name.value)
    void submit()
})

const v = computed(() => (data.value ? verdict(data.value) : undefined))
const wild = computed(() => (data.value ? wildcardCas(data.value) : 'any'))
const alert = computed(() => {
  const x = v.value
  if (!x)
    return undefined
  if (x.kind === 'any')
    return { type: 'warning' as const, text: t('tools.caa-checker.verdict.any') }
  if (x.kind === 'blocked')
    return { type: 'error' as const, text: t(`tools.caa-checker.verdict.${x.reason}`) }
  return { type: 'success' as const, text: t('tools.caa-checker.verdict.restricted', { cas: x.cas.join(', ') }) }
})
const rows = computed(() => (data.value
  ? [
      { label: t('tools.caa-checker.fields.relevant'), value: data.value.relevantName || t('common.none'), mono: true },
      { label: t('tools.caa-checker.fields.wildcard'), value: wild.value === 'any' ? t('tools.caa-checker.anyCa') : wild.value.length ? wild.value.join(', ') : t('tools.caa-checker.noCa'), noCopy: true },
      ...(data.value.iodef.length ? [{ label: t('tools.caa-checker.fields.iodef'), value: data.value.iodef.join(', '), mono: true }] : []),
    ]
  : []))
const steps = computed(() => (data.value?.steps ?? []).flatMap(s => s.records.length
  ? s.records.map(r => ({ name: s.name, rcode: s.rcode, record: `${r.flag} ${r.tag} "${r.value}"` }))
  : [{ name: s.name, rcode: s.rcode, record: '—' }]))
const columns = computed(() => [
  { key: 'name', title: t('tools.caa-checker.columns.name'), mono: true, width: 220 },
  { key: 'rcode', title: t('tools.caa-checker.columns.rcode'), width: 110 },
  { key: 'record', title: t('tools.caa-checker.columns.record'), mono: true, width: 340 },
])
const asText = () => [alert.value?.text, ...rows.value.map(r => `${r.label}: ${r.value}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="name" :label="t('tools.caa-checker.name')" placeholder="www.example.com" :validation="validation" monospace test-id="name" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!name || !validation.isValid" data-testid="run">
        {{ t('tools.caa-checker.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <NAlert v-if="alert" :type="alert.type" class="mb-3" data-testid="verdict">
      {{ alert.text }}
    </NAlert>
    <CKeyValueList :rows="rows" />
    <h3 class="mb-2 mt-4 text-sm font-semibold">
      {{ t('tools.caa-checker.steps') }}
    </h3>
    <ResultTable :columns="columns" :rows="steps" :actions="false" :label="t('tools.caa-checker.steps')" />
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="caa.json" />
    </div>
  </CCard>
</template>
