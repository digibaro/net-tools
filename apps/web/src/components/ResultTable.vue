<script setup lang="ts">
import type { DataTableColumns } from 'naive-ui'
import { TableExport } from '@vicons/tabler'
import { NA, NButton, NDataTable, NIcon } from 'naive-ui'
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { csvName, toCsv } from '@/utils/csv'
import { downloadText } from '@/utils/download'
import ResultActions from './ResultActions.vue'

export interface ResultColumn {
  key: string
  title: string
  sortable?: boolean
  mono?: boolean
  width?: number
  /** Row key holding a URL; the cell renders as an external link. */
  hrefKey?: string
  /** Row key holding an in-app path; the cell renders as a router link. */
  routeKey?: string
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
  render: c.hrefKey || c.routeKey
    ? (row: Row) => {
        const text = String(row[c.key] ?? '')
        const to = c.routeKey ? row[c.routeKey] : undefined
        if (typeof to === 'string' && to)
          return h(RouterLink, { to }, () => text)
        const href = c.hrefKey ? row[c.hrefKey] : undefined
        return typeof href === 'string' && href ? h(NA, { href, target: '_blank', rel: 'noopener noreferrer' }, () => text) : text
      }
    : undefined,
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

function downloadCsv() {
  const rows = data.value.map(r => props.columns.map(c => r[c.key]))
  downloadText(csvName(props.filename), toCsv(props.columns.map(c => c.title), rows), 'text/csv;charset=utf-8')
}

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
    <div v-if="rows.length" class="nt-no-print mt-3 flex flex-wrap gap-2">
      <ResultActions v-if="actions" :text="asText" :json="rows" :filename="filename" />
      <NButton size="small" secondary data-testid="download-csv" @click="downloadCsv">
        <template #icon>
          <NIcon aria-hidden="true" :component="TableExport" />
        </template>
        {{ t('common.downloadCsv') }}
      </NButton>
    </div>
  </div>
</template>
