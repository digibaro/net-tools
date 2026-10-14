<script setup lang="ts">
import type { ScanProbe, ScanSummary } from './tls-scanner.service'
import { NAlert, NButton, NFormItem, NInputNumber, NProgress, NTag } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'
import { ciphersByVersion } from './tls-scanner.service'

const { t, te } = useI18n()
const host = useQueryParam({ name: 'host', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 443 })
const validation = useValidation({ source: host, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const probes = ref<ScanProbe[]>([])
const summary = shallowRef<ScanSummary>()
const { status, error, start, stop } = useSse('/v1/tls/scan/stream', {
  probe: d => probes.value.push(d as ScanProbe),
  done: d => (summary.value = (d as { summary: ScanSummary }).summary),
})
const running = computed(() => status.value === 'running')
function run() {
  if (!host.value || !validation.value.isValid)
    return
  probes.value = []
  summary.value = undefined
  start({ host: host.value.trim(), port: port.value })
}
onMounted(() => {
  if (host.value)
    run()
})
const versions = computed(() => probes.value.filter(p => p.kind === 'version'))
const groups = computed(() => probes.value.filter(p => p.kind === 'group'))
const byVersion = computed(() => ciphersByVersion(probes.value))
const legacy = (v: string) => v === 'TLS 1.0' || v === 'TLS 1.1'
const ratingType = { good: 'success', fair: 'warning', weak: 'error' } as const
const finding = (c: string) => (te(`tools.tls-scanner.findings.${c}`) ? t(`tools.tls-scanner.findings.${c}`) : c)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="host" :label="t('tools.revocation-check.host')" placeholder="example.com" :validation="validation" monospace test-id="host" />
        <NFormItem :label="t('tools.tcp-ping.port')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.tcp-ping.port') }" />
        </NFormItem>
      </div>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="!host || !validation.isValid || running" data-testid="run">
          {{ t('tools.tls-scanner.run') }}
        </NButton>
        <NButton v-if="running" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.tls-scanner.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <NProgress v-if="running" type="line" :percentage="Math.min(95, probes.length * 2)" :show-indicator="false" :aria-label="t('tools.tls-scanner.run')" class="mb-4" />
  <template v-if="probes.length">
    <NAlert v-if="summary" :type="ratingType[summary.rating]" class="mb-4" data-testid="result">
      <strong>{{ t(`tools.tls-scanner.rating.${summary.rating}`) }}</strong>
      <ul v-if="summary.problems.length || summary.warnings.length" class="mb-0 mt-1 pl-5">
        <li v-for="p in summary.problems" :key="p">
          {{ finding(p) }}
        </li>
        <li v-for="w in summary.warnings" :key="w">
          {{ finding(w) }}
        </li>
      </ul>
    </NAlert>
    <CCard :title="t('tools.tls-scanner.versions')">
      <div class="flex flex-wrap gap-2">
        <NTag v-for="v in versions" :key="v.name" :type="v.supported ? (legacy(v.name) ? 'warning' : 'success') : 'default'" :bordered="false">
          {{ v.name }}: {{ v.supported ? t('common.yes') : t('common.no') }}
        </NTag>
      </div>
    </CCard>
    <CCard v-for="g in byVersion" :key="g.version" :title="t('tools.tls-scanner.ciphers', { version: g.version })">
      <ul class="m-0 list-none p-0 text-sm font-mono">
        <li v-for="c in g.ciphers" :key="c.name" class="flex flex-wrap items-center gap-2 py-0.5">
          <span class="break-all">{{ c.name }}</span>
          <NTag v-for="w in c.weak" :key="w" size="tiny" :bordered="false" :type="w === 'cbc' ? 'warning' : 'error'">
            {{ w }}
          </NTag>
        </li>
      </ul>
    </CCard>
    <CCard v-if="groups.length" :title="t('tools.tls-scanner.groups')">
      <div class="flex flex-wrap gap-2">
        <NTag v-for="g in groups" :key="g.name" :type="g.supported ? 'success' : 'default'" :bordered="false">
          {{ g.name }}: {{ g.supported ? t('common.yes') : t('common.no') }}
        </NTag>
      </div>
    </CCard>
    <ResultActions v-if="summary" :text="() => [`${host}: ${summary!.rating}`, `versions: ${summary!.versions.join(', ')}`, ...Object.entries(summary!.ciphers).map(([v, l]) => `${v}: ${l.join(', ')}`), `groups: ${summary!.groups.join(', ')}`].join('\n')" :json="summary" filename="tls-scan.json" />
  </template>
</template>
