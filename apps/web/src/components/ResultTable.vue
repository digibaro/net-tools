<script setup lang="ts">
import type { DataTableColumns } from 'naive-ui'
import { NDataTable } from 'naive-ui'
import { computed } from 'vue'
import ResultActions from './ResultActions.vue'

export interface ResultColumn {
  key: string
  title: string
  sortable?: boolean
  mono?: boolean
  width?: number
}

type Row = Record<string, string | number | boolean | null | undefined>

const props = withDefaults(defineProps<{
  columns: ResultColumn[]
  rows: Row[]
  filename?: string
  /** Show copy / download actions. */
  actions?: boolean
  maxHeight?: number
  rowKey?: string
}>(), { filename: 'result.json', actions: true })

const tableColumns = computed<DataTableColumns<Row>>(() => props.columns.map(c => ({
  key: c.key,
  title: c.title,
  width: c.width,
  className: c.mono ? 'font-mono' : undefined,
  sorter: c.sortable
    ? (a: Row, b: Row) => {
        const x = a[c.key]
        const y = b[c.key]
        if (typeof x === 'number' && typeof y === 'number')
          return x - y
        return String(x ?? '').localeCompare(String(y ?? ''), undefined, { numeric: true })
      }
    : undefined,
})))

function asText(): string {
  const header = props.columns.map(c => c.title).join('\t')
  const lines = props.rows.map(r => props.columns.map(c => String(r[c.key] ?? '')).join('\t'))
  return [header, ...lines].join('\n')
}
</script>

<template>
  <div>
    <NDataTable
      :columns="tableColumns"
      :data="rows"
      :max-height="maxHeight"
      :row-key="rowKey ? (r: Row) => String(r[rowKey!]) : undefined"
      size="small"
      :scroll-x="columns.length * 140"
      :bordered="false"
      striped
    />
    <div v-if="actions && rows.length" class="mt-3">
      <ResultActions :text="asText" :json="rows" :filename="filename" />
    </div>
  </div>
</template>
