<script setup lang="ts">
import type { StatusClass } from './http-status-codes.service'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { filterCodes, rfcUrl } from './http-status-codes.service'

const { t } = useI18n()
const query = useQueryParam({ name: 'q', defaultValue: '' })
const clsParam = useQueryParam({ name: 'class', defaultValue: 'all' })
const classes: StatusClass[] = ['all', '1', '2', '3', '4', '5']
const cls = computed<StatusClass>({
  get: () => (classes.includes(clsParam.value as StatusClass) ? clsParam.value as StatusClass : 'all'),
  set: (v) => {
    clsParam.value = v
  },
})
const rows = computed(() => filterCodes(query.value, cls.value).map(c => ({ ...c, reference: `RFC ${c.rfc} §${c.section}`, url: rfcUrl(c) })))
const k = (key: string) => t(`tools.http-status-codes.columns.${key}`)
const columns = computed(() => [
  { key: 'code', title: k('code'), sortable: true, width: 80 },
  { key: 'phrase', title: k('phrase'), width: 220 },
  { key: 'description', title: k('description'), width: 380 },
  { key: 'reference', title: k('reference'), hrefKey: 'url', width: 150 },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="query" :label="t('tools.http-status-codes.search')" placeholder="404, redirect, timeout…" test-id="input" />
    <NRadioGroup v-model:value="cls" name="status-class" size="small">
      <NRadioButton v-for="c in classes" :key="c" :value="c" :label="t(`tools.http-status-codes.classes.${c}`)" />
    </NRadioGroup>
  </CCard>
  <CCard :title="t('tools.http-status-codes.count', { count: rows.length })" data-testid="result">
    <ResultTable :columns="columns" :rows="rows" row-key="code" filename="http-status-codes.json" />
  </CCard>
</template>
