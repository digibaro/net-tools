<script setup lang="ts">
import type { DnsblCheck, DnsblSummary } from './blacklist-check.service'
import { NAlert, NButton, NProgress, NTag } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { sortChecks } from './blacklist-check.service'

const { t } = useI18n()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const checks = ref<DnsblCheck[]>([])
const total = ref(0)
const summary = shallowRef<DnsblSummary>()
const { status, error, start } = useSse('/v1/email/dnsbl/stream', {
  meta: d => (total.value = (d as { lists: number }).lists),
  list: d => checks.value.push(d as DnsblCheck),
  done: d => (summary.value = (d as { summary: DnsblSummary }).summary),
})
const running = computed(() => status.value === 'running')
function run() {
  if (!target.value.trim())
    return
  checks.value = []
  summary.value = undefined
  start({ target: target.value.trim() })
}
onMounted(() => {
  if (target.value)
    run()
})
const sorted = computed(() => sortChecks(checks.value))
const statusType = { listed: 'error', error: 'warning', clean: 'success' } as const
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="target" :label="t('tools.blacklist-check.target')" placeholder="192.0.2.25 or example.com" monospace test-id="target" />
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!target.trim() || running" data-testid="run">
        {{ t('tools.blacklist-check.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.blacklist-check.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <NProgress v-if="running" type="line" :percentage="total ? Math.round(checks.length / total * 100) : 0" :show-indicator="false" :aria-label="t('tools.blacklist-check.run')" class="mb-4" />
  <template v-if="checks.length">
    <NAlert v-if="summary" :type="summary.listed.length ? 'error' : 'success'" class="mb-4" data-testid="result">
      {{ summary.listed.length ? t('tools.blacklist-check.listed', { n: summary.listed.length, total: summary.checked, lists: summary.listed.join(', ') }) : t('tools.blacklist-check.clean', { total: summary.checked }) }}
      <template v-if="summary.errors">
        {{ t('tools.blacklist-check.errors', { n: summary.errors }) }}
      </template>
    </NAlert>
    <CCard :title="t('tools.blacklist-check.lists')">
      <ul class="m-0 list-none p-0">
        <li v-for="c in sorted" :key="c.zone + c.queried" class="flex flex-wrap items-center gap-2 py-1 text-sm">
          <NTag size="small" :bordered="false" :type="statusType[c.status]">
            {{ t(`tools.blacklist-check.status.${c.status}`) }}
          </NTag>
          <span>{{ c.name }}</span>
          <span class="text-xs nt-muted font-mono">{{ c.zone }}<template v-if="c.queried"> · {{ c.queried }}</template></span>
          <span v-if="c.codes.length" class="text-xs font-mono">{{ c.codes.join(' ') }}</span>
          <span v-if="c.reason" class="w-full break-all pl-2 text-xs">{{ c.reason }}</span>
          <span v-if="c.error" class="w-full pl-2 text-xs nt-muted">{{ c.error }}</span>
          <a v-if="c.status === 'listed' && c.delist" :href="c.delist" target="_blank" rel="noopener noreferrer" class="text-xs">{{ t('tools.blacklist-check.delist') }}</a>
        </li>
      </ul>
      <div v-if="summary" class="mt-3">
        <ResultActions :text="() => sorted.map(c => `${c.status}\t${c.name}\t${c.zone}\t${c.codes.join(' ')}`).join('\n')" :json="{ summary, checks: sorted }" filename="dnsbl.json" />
      </div>
    </CCard>
  </template>
</template>
