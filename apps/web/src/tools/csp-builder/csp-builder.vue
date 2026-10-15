<script setup lang="ts">
import type { Output } from './csp-builder.service'
import { NAlert, NButton, NFormItem, NInput, NSwitch } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CSelect from '@/ui/c-select.vue'
import { evaluate, newNonce, parseCsp, PRESETS, render } from './csp-builder.service'

const { t } = useI18n()
const policy = useQueryParam({ name: 'csp', defaultValue: PRESETS.strict })
const reportOnly = useQueryParam({ name: 'reportOnly', defaultValue: false })
const output = useQueryParam({ name: 'output', defaultValue: 'header' })
const findings = computed(() => evaluate(policy.value))
const directives = computed(() => parseCsp(policy.value).map((d, i) => ({ id: String(i), name: d.name, values: d.values.join(' ') || '—' })))
const rendered = computed(() => render(policy.value, output.value as Output, reportOnly.value))
const outputOptions = computed(() => (['header', 'nginx', 'apache', 'meta'] as const).map(v => ({ label: t(`tools.csp-builder.outputs.${v}`), value: v })))
const severityType = { bad: 'error', warn: 'warning', info: 'info' } as const
const order = { bad: 0, warn: 1, info: 2 }
const sorted = computed(() => [...findings.value].sort((a, b) => order[a.severity] - order[b.severity]))
const score = computed(() => (findings.value.some(f => f.severity === 'bad') ? 'bad' : findings.value.some(f => f.severity === 'warn') ? 'warn' : 'good'))
function preset(p: keyof typeof PRESETS) {
  policy.value = PRESETS[p]
}
function fillNonce() {
  const n = newNonce()
  policy.value = /'nonce-[^']*'/.test(policy.value) ? policy.value.replace(/'nonce-[^']*'/g, `'nonce-${n}'`) : policy.value.replace(/script-src/, `script-src 'nonce-${n}'`)
}
</script>

<template>
  <CCard>
    <div class="mb-3 flex flex-wrap gap-2">
      <NButton size="small" data-testid="preset-strict" @click="preset('strict')">
        {{ t('tools.csp-builder.presets.strict') }}
      </NButton>
      <NButton size="small" @click="preset('allowlist')">
        {{ t('tools.csp-builder.presets.allowlist') }}
      </NButton>
      <NButton size="small" @click="preset('lockdown')">
        {{ t('tools.csp-builder.presets.lockdown') }}
      </NButton>
      <NButton size="small" quaternary @click="fillNonce">
        {{ t('tools.csp-builder.nonce') }}
      </NButton>
    </div>
    <NFormItem :label="t('tools.csp-builder.policy')" :show-feedback="false" class="mb-3">
      <NInput v-model:value="policy" type="textarea" :autosize="{ minRows: 3, maxRows: 10 }" class="font-mono" :input-props="{ 'aria-label': t('tools.csp-builder.policy'), 'spellcheck': false }" data-testid="policy" />
    </NFormItem>
    <p class="m-0 text-xs nt-muted">
      {{ t('tools.csp-builder.help') }}
    </p>
  </CCard>
  <CCard :title="t('tools.csp-builder.evaluation')" data-testid="result">
    <NAlert :type="score === 'bad' ? 'error' : score === 'warn' ? 'warning' : 'success'" class="mb-3" data-testid="verdict">
      {{ t(`tools.csp-builder.score.${score}`) }}
    </NAlert>
    <NAlert v-for="(f, i) in sorted" :key="i" :type="severityType[f.severity]" class="mb-2" :bordered="false" data-testid="finding">
      <span v-if="f.directive" class="font-mono">{{ f.directive }}: </span>{{ t(`tools.csp-builder.findings.${f.id}`, { detail: f.detail }) }}
    </NAlert>
    <ResultTable class="mt-3" :columns="[{ key: 'name', title: t('tools.csp-builder.directive'), mono: true, width: 200 }, { key: 'values', title: t('tools.csp-builder.values'), mono: true, width: 480 }]" :rows="directives" row-key="id" :actions="false" :label="t('tools.csp-builder.directive')" />
  </CCard>
  <CCard :title="t('tools.csp-builder.output')">
    <div class="grid items-end gap-x-4 sm:grid-cols-[1fr_auto]">
      <CSelect v-model:value="output" :label="t('tools.csp-builder.format')" :options="outputOptions" test-id="output-format" />
      <label class="mb-3 flex items-center gap-2 text-sm">
        <NSwitch v-model:value="reportOnly" size="small" :disabled="output === 'meta'" />
        {{ t('tools.csp-builder.reportOnly') }}
      </label>
    </div>
    <div class="flex items-start gap-2">
      <pre class="m-0 min-w-0 flex-1 overflow-auto whitespace-pre-wrap break-all text-xs font-mono" tabindex="0" data-testid="output">{{ rendered }}</pre>
      <CopyButton :text="rendered" :label="t('common.copyText')" />
    </div>
    <p v-if="output === 'meta'" class="mb-0 text-xs nt-muted">
      {{ t('tools.csp-builder.metaNote') }}
    </p>
  </CCard>
</template>
