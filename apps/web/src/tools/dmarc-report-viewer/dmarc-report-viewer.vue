<script setup lang="ts">
import type { DmarcReport } from './dmarc-report-viewer.service'
import { Upload } from '@vicons/tabler'
import { NAlert, NButton, NIcon, NProgress, useThemeVars } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { fixed } from '@/utils/format'
import { parseReportXml, readReports, ReportError, summarize } from './dmarc-report-viewer.service'

const { t } = useI18n()
const theme = useThemeVars()
const fileInput = ref<HTMLInputElement>()
const pasted = ref('')
const reports = shallowRef<DmarcReport[]>([])
const errors = ref<string[]>([])
const dragging = ref(false)

function errorText(e: unknown, file: string) {
  return e instanceof ReportError ? t(`tools.dmarc-report-viewer.errors.${e.key}`, { file: e.file }) : t('tools.dmarc-report-viewer.errors.unreadable', { file })
}
async function load(files: FileList | File[]) {
  const out: DmarcReport[] = []
  const errs: string[] = []
  for (const f of files) {
    try {
      out.push(...readReports(f.name, new Uint8Array(await f.arrayBuffer())))
    }
    catch (e) {
      errs.push(errorText(e, f.name))
    }
  }
  reports.value = out
  errors.value = errs
}
function onFile(e: Event) {
  const files = (e.target as HTMLInputElement).files
  if (files?.length)
    void load(files)
}
function onDrop(e: DragEvent) {
  dragging.value = false
  if (e.dataTransfer?.files.length)
    void load(e.dataTransfer.files)
}
function onPaste() {
  if (!pasted.value.trim())
    return
  try {
    reports.value = [parseReportXml(pasted.value, t('tools.dmarc-report-viewer.pastedName'))]
    errors.value = []
  }
  catch (e) {
    reports.value = []
    errors.value = [errorText(e, t('tools.dmarc-report-viewer.pastedName'))]
  }
}

const summary = computed(() => reports.value.length ? summarize(reports.value) : undefined)
const utcDay = (x: Date) => x.toISOString().slice(0, 10)
const pct = (n: number) => `${fixed(n * 100, 1)} %`
const summaryRows = computed(() => {
  const s = summary.value!
  const r0 = reports.value[0]!
  return [
    { label: t('tools.dmarc-report-viewer.rows.domains'), value: s.domains.join(', '), mono: true },
    { label: t('tools.dmarc-report-viewer.rows.policy'), value: `p=${r0.policy.p} sp=${r0.policy.sp || r0.policy.p} pct=${r0.policy.pct || '100'} adkim=${r0.policy.adkim} aspf=${r0.policy.aspf}`, mono: true },
    { label: t('tools.dmarc-report-viewer.rows.period'), value: s.begin && s.end ? `${utcDay(s.begin)} – ${utcDay(s.end)} UTC` : '—' },
    { label: t('tools.dmarc-report-viewer.rows.reporters'), value: s.reporters.join(', ') },
    { label: t('tools.dmarc-report-viewer.rows.messages'), value: String(s.messages), mono: true },
    { label: t('tools.dmarc-report-viewer.rows.pass'), value: `${s.pass} (${pct(s.passRate)})`, mono: true },
    { label: t('tools.dmarc-report-viewer.rows.dispositions'), value: Object.entries(s.dispositions).map(([k, v]) => `${k}: ${v}`).join(', '), mono: true },
  ]
})
const rows = computed(() => (summary.value?.sources ?? []).map(s => ({
  ip: s.sourceIp,
  link: `/ip-to-asn?ip=${encodeURIComponent(s.sourceIp)}`,
  count: s.count,
  fail: s.fail,
  rate: s.count ? pct(s.pass / s.count) : '—',
  from: s.headerFrom.join(', '),
  dkim: s.dkim.join('; ') || '—',
  spf: s.spf.join('; ') || '—',
  disposition: Object.entries(s.dispositions).map(([k, v]) => `${k} ${v}`).join(', '),
})))
const c = (k: string) => t(`tools.dmarc-report-viewer.columns.${k}`)
</script>

<template>
  <CCard>
    <div
      class="mb-3 flex flex-col items-center gap-2 border-2 rounded-md border-dashed p-6 text-center"
      :style="{ borderColor: dragging ? theme.primaryColor : theme.borderColor }"
      data-testid="dropzone"
      @dragover.prevent="dragging = true" @dragleave="dragging = false" @drop.prevent="onDrop"
    >
      <input ref="fileInput" type="file" multiple accept=".xml,.gz,.zip,application/xml,application/gzip,application/zip" class="hidden" :aria-label="t('tools.dmarc-report-viewer.upload')" data-testid="file" @change="onFile">
      <NButton @click="fileInput?.click()">
        <template #icon>
          <NIcon aria-hidden="true" :component="Upload" />
        </template>
        {{ t('tools.dmarc-report-viewer.upload') }}
      </NButton>
      <span class="text-sm nt-muted">{{ t('tools.dmarc-report-viewer.drop') }}</span>
    </div>
    <CInputText v-model:value="pasted" :label="t('tools.dmarc-report-viewer.paste')" multiline :rows="4" monospace placeholder="<feedback>…</feedback>" test-id="xml" />
    <NButton size="small" :disabled="!pasted.trim()" data-testid="parse" @click="onPaste">
      {{ t('tools.dmarc-report-viewer.parse') }}
    </NButton>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.dmarc-report-viewer.note') }}
    </p>
  </CCard>
  <NAlert v-for="e in errors" :key="e" type="error" class="mb-4">
    {{ e }}
  </NAlert>
  <template v-if="summary">
    <CCard :title="t('tools.dmarc-report-viewer.summary', { n: reports.length })" data-testid="result">
      <NProgress type="line" :percentage="Math.round(summary.passRate * 100)" :status="summary.passRate >= 0.98 ? 'success' : summary.passRate >= 0.9 ? 'warning' : 'error'" :aria-label="t('tools.dmarc-report-viewer.rows.pass')" class="mb-3" />
      <CKeyValueList :rows="summaryRows" />
    </CCard>
    <CCard :title="t('tools.dmarc-report-viewer.sources', { n: rows.length })">
      <ResultTable :columns="[{ key: 'ip', title: c('ip'), mono: true, width: 150, routeKey: 'link' }, { key: 'count', title: c('count'), width: 90, sortable: true }, { key: 'fail', title: c('fail'), width: 80, sortable: true }, { key: 'rate', title: c('rate'), width: 90 }, { key: 'from', title: c('from'), width: 160 }, { key: 'dkim', title: 'DKIM', mono: true, width: 260 }, { key: 'spf', title: 'SPF', mono: true, width: 220 }, { key: 'disposition', title: c('disposition'), width: 160 }]" :rows="rows" row-key="ip" :actions="false" />
      <div class="mt-3">
        <ResultActions :text="() => rows.map(r => `${r.ip}\t${r.count}\t${r.fail}\t${r.from}\t${r.dkim}\t${r.spf}`).join('\n')" :json="reports" filename="dmarc-reports.json" />
      </div>
    </CCard>
  </template>
</template>
