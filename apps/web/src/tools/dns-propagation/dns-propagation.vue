<script setup lang="ts">
import type { DnsPropagation } from './dns-propagation.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { agreement, groupLabel, PROPAGATION_TYPES } from './dns-propagation.service'

const { t } = useI18n()
const name = useQueryParam({ name: 'name', defaultValue: '' })
const type = useQueryParam({ name: 'type', defaultValue: 'A' })
const validation = useValidation({ source: name, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const typeOptions = PROPAGATION_TYPES.map(v => ({ label: v, value: v }))
const { data, error, loading, run } = useApi<DnsPropagation>('/v1/dns/propagation')
const submit = () => name.value && validation.value.isValid && run({ name: name.value, type: type.value })
onMounted(() => {
  if (name.value)
    void submit()
})

const groups = computed(() => (data.value?.groups ?? []).map((g, i) => ({
  group: groupLabel(i),
  count: g.count,
  rcode: g.rcode,
  answers: g.answers.join('\n') || '—',
})))
const rows = computed(() => (data.value?.rows ?? []).map(r => ({
  resolver: r.provider,
  address: r.address,
  location: r.location,
  group: r.error ? t('tools.dns-propagation.failed') : groupLabel(r.group),
  answers: r.error ? '' : r.answers.join('\n') || r.rcode,
  ttl: r.ttl ?? '—',
  rtt: r.error ? '—' : `${r.rttMs} ms`,
})))
const k = (key: string) => t(`tools.dns-propagation.columns.${key}`)
const groupColumns = computed(() => [
  { key: 'group', title: k('group'), width: 70 },
  { key: 'count', title: k('count'), width: 100 },
  { key: 'rcode', title: k('rcode'), width: 110 },
  { key: 'answers', title: k('answers'), mono: true, width: 360 },
])
const rowColumns = computed(() => [
  { key: 'resolver', title: k('resolver'), width: 170, sortable: true },
  { key: 'address', title: k('address'), mono: true, width: 150 },
  { key: 'location', title: k('location'), width: 130 },
  { key: 'group', title: k('group'), width: 80, sortable: true },
  { key: 'answers', title: k('answers'), mono: true, width: 300 },
  { key: 'ttl', title: k('ttl'), width: 80 },
  { key: 'rtt', title: k('rtt'), width: 90 },
])
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_9rem]">
        <CInputText v-model:value="name" :label="t('tools.dns-lookup.name')" placeholder="example.com" :validation="validation" monospace test-id="name" />
        <CSelect v-model:value="type" :label="t('tools.dns-lookup.type')" :options="typeOptions" />
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!name || !validation.isValid" data-testid="run">
        {{ t('tools.dns-propagation.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="data.consensus ? 'success' : 'warning'" class="mb-4" data-testid="consensus">
      {{ data.consensus ? t('tools.dns-propagation.consensus') : t('tools.dns-propagation.split', { groups: data.groups.length, agreement: agreement(data) }) }}
    </NAlert>
    <CCard :title="t('tools.dns-propagation.groups')" data-testid="groups">
      <ResultTable :columns="groupColumns" :rows="groups" :actions="false" class="whitespace-pre-line" :label="t('tools.dns-propagation.groups')" />
    </CCard>
    <CCard :title="t('tools.dns-propagation.resolvers')" data-testid="result">
      <ResultTable :columns="rowColumns" :rows="rows" class="whitespace-pre-line" filename="dns-propagation.json" />
    </CCard>
  </template>
</template>
