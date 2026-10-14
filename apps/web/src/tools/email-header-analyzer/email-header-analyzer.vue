<script setup lang="ts">
import { NAlert, NTag } from 'naive-ui'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { authResults, dkimSignatures, formatDelay, hops, parseHeaders, summary } from './email-header-analyzer.service'

const { t } = useI18n()
// Headers often contain personal data, so they are deliberately not stored in the URL.
const raw = ref('')
const headers = computed(() => parseHeaders(raw.value))
const timeline = computed(() => hops(headers.value))
const auth = computed(() => authResults(headers.value))
const dkim = computed(() => dkimSignatures(headers.value))
const summaryRows = computed(() => summary(headers.value).map(h => ({ label: h.name, value: h.value })))
const totalDelay = computed(() => {
  const dated = timeline.value.filter(h => h.date)
  return dated.length >= 2 ? Math.round((dated.at(-1)!.date!.getTime() - dated[0]!.date!.getTime()) / 1000) : undefined
})
const slowest = computed(() => Math.max(0, ...timeline.value.map(h => h.delay ?? 0)))
const hopRows = computed(() => timeline.value.map(h => ({
  index: h.index,
  delay: h.delay === undefined ? '' : formatDelay(h.delay),
  from: h.from,
  by: h.by,
  with: h.with,
  date: h.date ? h.date.toISOString().replace('T', ' ').replace('.000Z', ' UTC') : '',
})))
const resultType = (r: string) => (r === 'pass' ? 'success' : r === 'none' || r === 'neutral' ? 'default' : r === 'softfail' || r === 'temperror' ? 'warning' : 'error')
const c = (key: string) => t(`tools.email-header-analyzer.columns.${key}`)
const asText = () => timeline.value.map(h => `${h.index}\t${h.delay === undefined ? '' : formatDelay(h.delay)}\t${h.from} → ${h.by}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText v-model:value="raw" :label="t('tools.email-header-analyzer.input')" :placeholder="t('tools.email-header-analyzer.placeholder')" multiline :rows="10" monospace test-id="input" />
    <p class="m-0 text-xs nt-muted">
      {{ t('tools.email-header-analyzer.privacy') }}
    </p>
  </CCard>
  <template v-if="headers.length">
    <CCard v-if="summaryRows.length" :title="t('tools.email-header-analyzer.summary')" data-testid="summary">
      <CKeyValueList :rows="summaryRows" />
    </CCard>
    <CCard :title="t('tools.email-header-analyzer.auth')" data-testid="auth">
      <div v-if="auth.length" class="flex flex-col gap-2">
        <div v-for="(a, i) in auth" :key="i" class="flex flex-wrap items-center gap-2 text-sm">
          <NTag :type="resultType(a.result)" size="small" :bordered="false">
            {{ a.method.toUpperCase() }} {{ a.result }}
          </NTag>
          <span class="break-all text-xs nt-muted font-mono">{{ a.detail }}</span>
        </div>
      </div>
      <p v-else class="m-0 text-sm nt-muted">
        {{ t('tools.email-header-analyzer.noAuth') }}
      </p>
      <p v-for="(d, i) in dkim" :key="`d${i}`" class="m-0 mt-2 text-sm">
        {{ t('tools.email-header-analyzer.dkimSig', { domain: d.domain, selector: d.selector, alg: d.algorithm }) }}
        <RouterLink :to="{ path: '/dkim-checker', query: { domain: d.domain, selector: d.selector } }">
          {{ t('tools.email-header-analyzer.checkKey') }}
        </RouterLink>
      </p>
    </CCard>
    <CCard :title="t('tools.email-header-analyzer.hops', { n: timeline.length })" data-testid="result">
      <NAlert v-if="totalDelay !== undefined" :type="slowest > 300 ? 'warning' : 'info'" class="mb-3">
        {{ t('tools.email-header-analyzer.total', { total: formatDelay(totalDelay) }) }}
        <template v-if="slowest > 300">
          {{ t('tools.email-header-analyzer.slow', { delay: formatDelay(slowest) }) }}
        </template>
      </NAlert>
      <ResultTable v-if="hopRows.length" :columns="[{ key: 'index', title: '#', width: 50 }, { key: 'delay', title: c('delay'), width: 110 }, { key: 'from', title: c('from'), mono: true, width: 280 }, { key: 'by', title: c('by'), mono: true, width: 240 }, { key: 'with', title: c('with'), width: 100 }, { key: 'date', title: c('date'), mono: true, width: 200 }]" :rows="hopRows" row-key="index" filename="email-hops.json" />
      <p v-else class="m-0 text-sm nt-muted">
        {{ t('tools.email-header-analyzer.noHops') }}
      </p>
      <div class="mt-3">
        <ResultActions :text="asText" :json="{ summary: summaryRows, auth, dkim, hops: hopRows }" filename="email-headers.json" />
      </div>
    </CCard>
  </template>
</template>
