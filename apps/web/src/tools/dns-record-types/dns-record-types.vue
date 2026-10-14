<script setup lang="ts">
import { NFormItem, NSwitch } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { filterTypes } from './dns-record-types.service'

const { t } = useI18n()
const query = useQueryParam({ name: 'q', defaultValue: '' })
const common = useQueryParam({ name: 'common', defaultValue: true })
const rows = computed(() => filterTypes(query.value, common.value).map(r => ({
  type: r.type,
  value: r.value,
  summary: r.summary,
  example: r.example,
  reference: r.rfc[0] ?? '',
  url: r.rfc[0] ? `https://www.rfc-editor.org/rfc/rfc${r.rfc[0].slice(4)}` : '',
})))
const k = (key: string) => t(`tools.dns-record-types.columns.${key}`)
const columns = computed(() => [
  { key: 'type', title: k('type'), width: 110, mono: true, sortable: true },
  { key: 'value', title: k('value'), width: 80, sortable: true },
  { key: 'summary', title: k('summary'), width: 300 },
  { key: 'example', title: k('example'), width: 420, mono: true },
  { key: 'reference', title: k('reference'), width: 110, hrefKey: 'url' },
])
</script>

<template>
  <CCard>
    <div class="flex flex-wrap items-end gap-x-6">
      <div class="min-w-64 flex-1">
        <CInputText v-model:value="query" :label="t('tools.dns-record-types.search')" placeholder="MX, 257, dnssec…" test-id="input" />
      </div>
      <NFormItem :label="t('tools.dns-record-types.commonOnly')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="common" :aria-label="t('tools.dns-record-types.commonOnly')" />
      </NFormItem>
    </div>
  </CCard>
  <CCard :title="t('tools.dns-record-types.count', { count: rows.length })" data-testid="result">
    <ResultTable :columns="columns" :rows="rows" row-key="type" filename="dns-record-types.json" />
  </CCard>
  <p class="text-xs nt-muted">
    {{ t('tools.dns-record-types.source') }}
  </p>
</template>
