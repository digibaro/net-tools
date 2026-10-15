<script setup lang="ts">
import type { NeighbourGraph } from './asn-neighbour-graph.service'
import { NButton, useThemeVars } from 'naive-ui'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { asnPath, parseAsn } from '@/utils/bgp'
import { GRAPH_W, layout, MAX_SHOWN, shortName } from './asn-neighbour-graph.service'

const { t } = useI18n()
const theme = useThemeVars()
const router = useRouter()
const asn = useQueryParam({ name: 'asn', defaultValue: '' })
const input = ref(asn.value)
const validation = useValidation({ source: input, rules: [{ validator: (v: string) => parseAsn(v) !== undefined, message: t('bgp.invalidAsn') }] })
const { data, error, loading, run } = useApi<NeighbourGraph>('/v1/bgp/neighbours')
function submit() {
  const n = parseAsn(input.value)
  if (n === undefined)
    return
  if (asn.value === `AS${n}`)
    void run({}, `/AS${n}`)
  asn.value = `AS${n}`
}
watch(asn, (v) => {
  input.value = v
  const n = parseAsn(v)
  if (n !== undefined)
    void run({}, `/AS${n}`)
}, { immediate: true })
const g = computed(() => (data.value ? layout(data.value) : undefined))
const open = (n: number) => router.push({ query: { asn: `AS${n}` } })
const tableRows = computed(() => (data.value
  ? [
      ...data.value.upstreams.map(n => ({ ...n, rel: t('tools.asn-neighbour-graph.upstream') })),
      ...data.value.downstreams.map(n => ({ ...n, rel: t('tools.asn-neighbour-graph.downstream') })),
      ...data.value.uncertain.map(n => ({ ...n, rel: t('tools.asn-neighbour-graph.uncertain') })),
    ].map(n => ({ id: `${n.rel}${n.asn}`, asn: `AS${n.asn}`, path: asnPath(n.asn), name: n.name || '—', rel: n.rel, power: n.power }))
  : []))
const k = (key: string) => t(`tools.asn-neighbour-graph.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="input" :label="t('bgp.asn')" placeholder="AS3333" :validation="validation" monospace test-id="asn" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!validation.isValid" data-testid="run">
        {{ t('tools.asn-neighbour-graph.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data && g">
    <CCard :title="`AS${data.asn} ${data.name}`" data-testid="result">
      <p class="mt-0 text-sm nt-muted">
        {{ t('tools.asn-neighbour-graph.summary', { up: data.upstreamTotal, down: data.downstreamTotal, unsure: data.uncertainTotal, shown: MAX_SHOWN }) }}
      </p>
      <div class="overflow-x-auto" tabindex="0" :aria-label="t('tools.asn-neighbour-graph.graph')">
        <svg :viewBox="`0 0 ${GRAPH_W} ${g.height}`" class="block min-w-[640px] w-full" role="img" :aria-label="t('tools.asn-neighbour-graph.graph')" data-testid="graph">
          <text x="140" y="12" text-anchor="middle" font-size="11" :fill="theme.textColor3">{{ t('tools.asn-neighbour-graph.upstreams') }}</text>
          <text :x="GRAPH_W - 140" y="12" text-anchor="middle" font-size="11" :fill="theme.textColor3">{{ t('tools.asn-neighbour-graph.downstreams') }}</text>
          <line v-for="p in g.nodes" :key="`l${p.side}${p.node.asn}`" :x1="p.side === 'up' ? p.x + 110 : p.x - 110" :y1="p.y" :x2="g.cx" :y2="g.cy" :stroke="theme.dividerColor" stroke-width="1.5" />
          <g v-for="p in g.nodes" :key="`n${p.side}${p.node.asn}`" class="cursor-pointer" role="link" tabindex="0" @click="open(p.node.asn)" @keydown.enter="open(p.node.asn)">
            <title>AS{{ p.node.asn }} {{ p.node.name }}</title>
            <rect :x="p.x - 110" :y="p.y - 12" width="220" height="24" rx="5" :fill="theme.cardColor" :stroke="p.side === 'up' ? theme.infoColor : theme.successColor" />
            <text :x="p.x" :y="p.y + 4" text-anchor="middle" font-size="11" :fill="theme.textColor1">AS{{ p.node.asn }} {{ shortName(p.node.name) }}</text>
          </g>
          <rect :x="g.cx - 120" :y="g.cy - 16" width="240" height="32" rx="6" :fill="theme.primaryColor" />
          <text :x="g.cx" :y="g.cy + 5" text-anchor="middle" font-size="12" font-weight="600" :fill="theme.baseColor">AS{{ data.asn }} {{ shortName(data.name) }}</text>
        </svg>
      </div>
    </CCard>
    <CCard :title="t('tools.asn-neighbour-graph.all')">
      <ResultTable :columns="[{ key: 'asn', title: 'ASN', width: 100, routeKey: 'path' }, { key: 'name', title: k('name'), width: 280 }, { key: 'rel', title: k('rel'), width: 120 }, { key: 'power', title: k('power'), width: 90 }]" :rows="tableRows" row-key="id" filename="asn-neighbours.json" />
    </CCard>
  </template>
</template>
