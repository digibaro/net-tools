<script setup lang="ts">
import type { EdnsCompResult, EdnsServer } from './edns-compliance.service'
import { NAlert, NButton, NTag } from 'naive-ui'
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { isServer } from '../open-resolver-test/open-resolver-test.service'
import { grade } from './edns-compliance.service'

const { t } = useI18n()
const zone = useQueryParam({ name: 'zone', defaultValue: '' })
const server = useQueryParam({ name: 'server', defaultValue: '' })
const validation = useValidation({ source: zone, rules: [{ validator: (s: string) => !s || isValidQueryName(s), message: t('tools.dns-lookup.invalidName') }] })
const serverValidation = useValidation({ source: server, rules: [{ validator: (s: string) => !s || isServer(s), message: t('tools.open-resolver-test.invalid') }] })
const { data, error, loading, run } = useApi<EdnsCompResult>('/v1/dns/edns-compliance')
const submit = () => zone.value && validation.value.isValid && serverValidation.value.isValid && run({ zone: zone.value.trim(), server: server.value.trim() || undefined })
onMounted(() => {
  if (zone.value)
    void submit()
})
const gradeType = { ok: 'success', minor: 'warning', broken: 'error', unreachable: 'default' } as const
const rows = (s: EdnsServer) => s.checks.map(c => ({ id: c.id, test: t(`tools.edns-compliance.checks.${c.id}`), result: c.pass ? t('tools.edns-compliance.pass') : t('tools.edns-compliance.fail'), detail: c.detail || '—' }))
const k = (key: string) => t(`tools.edns-compliance.columns.${key}`)
const asText = () => (data.value?.servers ?? []).map(s => `${s.name} (${s.addr}): ${s.passed}/${s.checks.length} ${s.checks.filter(c => !c.pass).map(c => c.id).join(' ')}`).join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="zone" :label="t('tools.edns-compliance.zone')" placeholder="example.com" :validation="validation" monospace test-id="zone" />
        <CInputText v-model:value="server" :label="t('tools.edns-compliance.server')" :placeholder="t('tools.edns-compliance.serverPlaceholder')" :validation="serverValidation" monospace test-id="server" />
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!zone || !validation.isValid" data-testid="run">
        {{ t('tools.edns-compliance.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.edns-compliance.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-for="s in data?.servers ?? []" :key="s.name + s.addr" :title="s.addr ? `${s.name} (${s.addr})` : s.name" data-testid="result">
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <NTag :type="gradeType[grade(s)]" :bordered="false">
        {{ t(`tools.edns-compliance.grade.${grade(s)}`) }}
      </NTag>
      <span v-if="!s.error" class="text-sm">{{ t('tools.edns-compliance.passed', { n: s.passed, total: s.checks.length }) }}</span>
      <span v-if="s.nsid" class="text-xs nt-muted font-mono">NSID {{ s.nsid }}</span>
      <span v-if="s.cookie" class="text-xs nt-muted">{{ t('tools.edns-compliance.cookie') }}</span>
    </div>
    <NAlert v-if="s.error" type="warning">
      {{ s.error }}
    </NAlert>
    <ResultTable v-else :columns="[{ key: 'test', title: k('test'), width: 280 }, { key: 'result', title: k('result'), width: 90 }, { key: 'detail', title: k('detail'), width: 320 }]" :rows="rows(s)" row-key="id" :actions="false" :label="s.name" />
  </CCard>
  <div v-if="data?.servers.length" class="mb-4">
    <ResultActions :text="asText" :json="data" filename="edns-compliance.json" />
  </div>
</template>
