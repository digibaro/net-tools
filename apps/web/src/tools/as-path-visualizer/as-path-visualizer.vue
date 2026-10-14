<script setup lang="ts">
import { NAlert, NTag, useThemeVars } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { buildGraph, parsePaths } from './as-path-visualizer.service'

const { t } = useI18n()
const theme = useThemeVars()
const input = useQueryParam({ name: 'paths', defaultValue: '3356 1299 13335\n174 13335 13335 13335\n6939 1299 13335\n2914 64512 13335' })
const paths = computed(() => parsePaths(input.value))
const graph = computed(() => buildGraph(paths.value))

// Layout constants (SVG user units).
const COL = 170
const ROW = 64
const NW = 120
const NH = 36
const PAD = 12
const width = computed(() => graph.value.layers * COL - (COL - NW) + 2 * PAD)
const height = computed(() => Math.max(1, ...graph.value.nodes.map(n => n.row + 1)) * ROW - (ROW - NH) + 2 * PAD + 14)
const pos = computed(() => new Map(graph.value.nodes.map(n => [n.asn, { x: PAD + (graph.value.layers - 1 - n.layer) * COL, y: PAD + n.row * ROW }])))
function edgePath(from: number, to: number) {
  const a = pos.value.get(from)!
  const b = pos.value.get(to)!
  const x1 = a.x + NW
  const y1 = a.y + NH / 2
  const x2 = b.x
  const y2 = b.y + NH / 2
  const mx = (x1 + x2) / 2
  return `M${x1},${y1} C${mx},${y1} ${mx},${y2} ${x2},${y2}`
}
const stroke = (cls: string, origin: boolean) => cls === 'private' ? theme.value.warningColor : cls !== 'public' ? theme.value.errorColor : origin ? theme.value.primaryColor : theme.value.borderColor

const rows = computed(() => paths.value.map((p, i) => ({
  id: i,
  path: p.raw,
  length: p.length,
  origin: p.origin !== undefined ? `AS${p.origin}` : 'AS_SET',
  notes: [
    ...p.prepends.map(x => t(x.atOrigin ? 'tools.as-path-visualizer.notes.originPrepend' : 'tools.as-path-visualizer.notes.prepend', { asn: x.asn, n: x.count })),
    ...p.loops.map(a => t('tools.as-path-visualizer.notes.loop', { asn: a })),
    ...p.privateAsns.map(a => t('tools.as-path-visualizer.notes.private', { asn: a })),
    ...p.reservedAsns.map(a => t('tools.as-path-visualizer.notes.reserved', { asn: a })),
    ...(p.hops.some(h => h.set) ? [t('tools.as-path-visualizer.notes.asSet')] : []),
  ].join('; ') || '—',
})))
const shortest = computed(() => paths.value.length ? Math.min(...paths.value.map(p => p.length)) : 0)
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.as-path-visualizer.input')" multiline :rows="6" monospace test-id="input" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.as-path-visualizer.hint') }}
    </p>
  </CCard>
  <CCard v-if="paths.length" :title="t('tools.as-path-visualizer.graph', { n: paths.length })" data-testid="result">
    <div class="overflow-x-auto">
      <svg :viewBox="`0 0 ${width} ${height}`" :width="width" :height="height" role="img" :aria-label="t('tools.as-path-visualizer.graphLabel')" class="block" data-testid="graph">
        <path v-for="e in graph.edges" :key="`${e.from}-${e.to}`" :d="edgePath(e.from, e.to)" fill="none" :stroke="theme.textColor3" :stroke-width="Math.min(1 + e.paths, 5)" />
        <g v-for="n in graph.nodes" :key="n.asn" :transform="`translate(${pos.get(n.asn)!.x},${pos.get(n.asn)!.y})`">
          <rect :width="NW" :height="NH" rx="6" :fill="theme.cardColor" :stroke="stroke(n.cls, n.isOrigin)" :stroke-width="n.isOrigin ? 2.5 : 1.5" />
          <text :x="NW / 2" :y="NH / 2 + 4" text-anchor="middle" :fill="theme.textColor1" font-size="13" font-family="ui-monospace, monospace">{{ n.label }}</text>
          <text v-if="n.prepend > 1" :x="NW - 4" y="-3" text-anchor="end" :fill="theme.warningColor" font-size="11">×{{ n.prepend }}</text>
        </g>
      </svg>
    </div>
    <div class="mt-2 flex flex-wrap gap-2 text-xs">
      <NTag size="small" :bordered="false" type="primary">
        {{ t('tools.as-path-visualizer.legend.origin') }}
      </NTag>
      <NTag size="small" :bordered="false" type="warning">
        {{ t('tools.as-path-visualizer.legend.private') }}
      </NTag>
      <NTag size="small" :bordered="false" type="error">
        {{ t('tools.as-path-visualizer.legend.reserved') }}
      </NTag>
      <span class="nt-muted">{{ t('tools.as-path-visualizer.legend.prepend') }}</span>
    </div>
  </CCard>
  <CCard v-if="paths.length" :title="t('tools.as-path-visualizer.analysis')">
    <NAlert v-if="paths.length > 1" type="info" class="mb-3">
      {{ t('tools.as-path-visualizer.shortest', { n: shortest }) }}
    </NAlert>
    <ResultTable :columns="[{ key: 'path', title: t('tools.as-path-visualizer.columns.path'), mono: true, width: 280 }, { key: 'length', title: t('tools.as-path-visualizer.columns.length'), width: 80 }, { key: 'origin', title: t('tools.as-path-visualizer.columns.origin'), mono: true, width: 110 }, { key: 'notes', title: t('tools.as-path-visualizer.columns.notes'), width: 360 }]" :rows="rows" row-key="id" :actions="false" />
    <div class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.path}\tlen=${r.length}\t${r.notes}`).join('\n')" :json="paths" filename="as-paths.json" />
    </div>
  </CCard>
</template>
