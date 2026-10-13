<script setup lang="ts">
import type { RdapRecord } from './rdap-lookup.service'
import { NButton } from 'naive-ui'
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
import { day, isRegistryQuery } from './rdap-lookup.service'

const { t } = useI18n()
const query = useQueryParam({ name: 'query', defaultValue: '' })
const validation = useValidation({ source: query, rules: [{ validator: isRegistryQuery, message: t('registry.invalidQuery') }] })
const { data, error, loading, run } = useApi<RdapRecord>('/v1/rdap')
const submit = () => query.value && validation.value.isValid && run({ query: query.value.trim() })
onMounted(() => {
  if (query.value)
    void submit()
})
const rows = computed(() => {
  const d = data.value
  if (!d)
    return []
  const k = (key: string) => t(`registry.fields.${key}`)
  return [
    { label: k('name'), value: d.name, mono: true },
    { label: k('handle'), value: d.handle, mono: true },
    { label: k('registrar'), value: d.registrar },
    { label: k('range'), value: d.range, mono: true },
    { label: k('type'), value: d.type },
    { label: k('country'), value: d.country },
    { label: k('status'), value: d.status.join(', ') },
    { label: k('nameservers'), value: d.nameservers.join(', '), mono: true },
    ...d.events.map(e => ({ label: e.action, value: day(e.date), mono: true })),
    { label: k('server'), value: d.server, mono: true },
  ].filter(r => r.value)
})
const entities = computed(() => (data.value?.entities ?? []).map(e => ({ roles: e.roles.join(', '), name: e.name || e.org, email: e.email, country: e.country, handle: e.handle })))
const c = (key: string) => t(`tools.rdap-lookup.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="query" :label="t('registry.query')" placeholder="example.com, 8.8.8.8, AS13335" :validation="validation" monospace test-id="query" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!query || !validation.isValid" data-testid="run">
        {{ t('tools.rdap-lookup.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="t('common.result')" data-testid="result">
      <CKeyValueList :rows="rows" />
      <div class="mt-3">
        <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="data" filename="rdap.json" />
      </div>
    </CCard>
    <CCard v-if="entities.length" :title="t('tools.rdap-lookup.entities')">
      <ResultTable :columns="[{ key: 'roles', title: c('roles'), width: 160 }, { key: 'name', title: c('name'), width: 220 }, { key: 'email', title: c('email'), mono: true, width: 220 }, { key: 'country', title: c('country'), width: 80 }, { key: 'handle', title: c('handle'), mono: true, width: 140 }]" :rows="entities" filename="rdap-entities.json" />
    </CCard>
  </template>
</template>
