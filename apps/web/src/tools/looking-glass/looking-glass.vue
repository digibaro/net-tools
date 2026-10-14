<script setup lang="ts">
import type { LookingGlass } from './looking-glass.service'
import { NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isPrefixOrIp, pathCounts } from './looking-glass.service'

const { t } = useI18n()
const prefix = useQueryParam({ name: 'prefix', defaultValue: '' })
const validation = useValidation({ source: prefix, rules: [{ validator: isPrefixOrIp, message: t('bgp.invalidPrefix') }] })
const { data, error, loading, run } = useApi<LookingGlass>('/v1/lookingglass')
const submit = () => prefix.value && validation.value.isValid && run({ prefix: prefix.value.trim() })
onMounted(() => {
  if (prefix.value)
    void submit()
})
const routes = computed(() => (data.value?.routes ?? []).map((r, i) => ({ id: i, collector: `${r.collector} (${r.location})`, peer: r.peer, path: r.asPath, community: r.community, nextHop: r.nextHop, updated: r.updated })))
const c = (key: string) => t(`bgp.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="prefix" :label="t('bgp.prefix')" placeholder="1.1.1.0/24" :validation="validation" monospace test-id="prefix" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!prefix || !validation.isValid" data-testid="run">
        {{ t('bgp.lookUp') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="t('tools.looking-glass.summary', { prefix: data.prefix })" data-testid="result">
      <p class="mt-0 text-sm">
        {{ t('tools.looking-glass.seen', { routes: data.total, collectors: data.collectors, origins: data.origins.map(o => `AS${o}`).join(', ') || '—' }) }}
      </p>
      <ResultTable :columns="[{ key: 'path', title: c('asPath'), mono: true, width: 320 }, { key: 'peers', title: c('peers'), width: 90, sortable: true }]" :rows="pathCounts(data)" row-key="path" :max-height="320" :actions="false" />
    </CCard>
    <CCard v-if="routes.length" :title="t('tools.looking-glass.routes')">
      <ResultTable :columns="[{ key: 'collector', title: c('collector'), width: 220 }, { key: 'peer', title: c('peer'), mono: true, width: 180 }, { key: 'path', title: c('asPath'), mono: true, width: 240 }, { key: 'community', title: c('community'), mono: true, width: 220 }, { key: 'updated', title: c('updated'), mono: true, width: 170 }]" :rows="routes" row-key="id" :max-height="480" filename="looking-glass.json" />
      <p v-if="data.total > routes.length" class="mb-0 text-xs nt-muted">
        {{ t('tools.looking-glass.truncated', { shown: routes.length, total: data.total }) }}
      </p>
    </CCard>
  </template>
</template>
