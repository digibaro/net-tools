<script setup lang="ts">
import type { DataTableColumns } from 'naive-ui'
import { NDataTable } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from './ResultActions.vue'

export interface ResultColumn {
  key: string
  title: string
  sortable?: boolean
  mono?: boolean
  width?: number
}

type Row = Record<string, unknown>

const props = withDefaults(defineProps<{
  columns: ResultColumn[]
  /** Any plain objects; values are shown via String(). */
  rows: object[]
  filename?: string
  /** Show copy / download actions. */
  actions?: boolean
  maxHeight?: number
  /** Accessible name of the scrollable region. */
  label?: string
  rowKey?: string
}>(), { filename: 'result.json', actions: true })

const { t } = useI18n()
const data = computed(() => props.rows as Row[])

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

// Below this width the region scrolls horizontally instead of squeezing columns.
const minWidth = computed(() => props.columns.reduce((sum, c) => sum + (c.width ?? 130), 0))

function asText(): string {
  const header = props.columns.map(c => c.title).join('\t')
  const lines = data.value.map(r => props.columns.map(c => String(r[c.key] ?? '')).join('\t'))
  return [header, ...lines].join('\n')
}
</script>

<template>
  <div>
    <!-- Our own scroll region: keyboard-focusable and labelled (axe scrollable-region-focusable). -->
    <div
      class="overflow-auto"
      :style="maxHeight ? { maxHeight: `${maxHeight}px` } : undefined"
      tabindex="0"
      role="region"
      :aria-label="label ?? t('common.result')"
    >
      <NDataTable
        :columns="tableColumns"
        :data="data"
        :row-key="rowKey ? (r: Row) => String(r[rowKey!]) : undefined"
        size="small"
        :bordered="false"
        striped
        :style="{ minWidth: `${minWidth}px` }"
      />
    </div>
    <div v-if="actions && rows.length" class="mt-3">
      <ResultActions :text="asText" :json="rows" :filename="filename" />
    </div>
  </div>
</template>
