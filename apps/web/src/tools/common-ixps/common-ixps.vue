<script setup lang="ts">
import type { CommonPeering } from './common-ixps.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { parseAsn } from '@/utils/bgp'
import { ports } from './common-ixps.service'

const { t } = useI18n()
const a = useQueryParam({ name: 'a', defaultValue: '' })
const b = useQueryParam({ name: 'b', defaultValue: '' })
const rule = [{ validator: (v: string) => !v || parseAsn(v) !== undefined, message: t('bgp.invalidAsn') }]
const va = useValidation({ source: a, rules: rule })
const vb = useValidation({ source: b, rules: rule })
const ready = computed(() => parseAsn(a.value) !== undefined && parseAsn(b.value) !== undefined)
const { data, error, loading, run } = useApi<CommonPeering>('/v1/peeringdb/common')
const submit = () => ready.value && run({ a: a.value.trim(), b: b.value.trim() })
onMounted(() => {
  if (ready.value)
    void submit()
})
const name = (s: CommonPeering['a']) => `AS${s.asn}${s.name ? ` ${s.name}` : ''}`
const ixRows = computed(() => (data.value?.ixs ?? []).map(x => ({ id: String(x.ixId), name: x.name, a: ports(x.a), b: ports(x.b), rs: x.bothRs ? t('common.yes') : t('common.no') })))
const facRows = computed(() => (data.value?.facilities ?? []).map(f => ({ id: String(f.facId), name: f.name, where: [f.city, f.country].filter(Boolean).join(', ') })))
const missing = computed(() => (data.value ? [data.value.a, data.value.b].filter(s => !s.found) : []))
const k = (key: string) => t(`tools.common-ixps.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="a" :label="t('tools.common-ixps.a')" placeholder="AS3333" :validation="va" monospace test-id="a" />
        <CInputText v-model:value="b" :label="t('tools.common-ixps.b')" placeholder="AS13335" :validation="vb" monospace test-id="b" />
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ready" data-testid="run">
        {{ t('tools.common-ixps.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-for="s in missing" :key="s.asn" type="warning" class="mb-4">
      {{ t('tools.common-ixps.notFound', { asn: s.asn }) }}
    </NAlert>
    <CCard v-if="!missing.length" :title="t('tools.common-ixps.title2', { a: name(data.a), b: name(data.b) })" data-testid="result">
      <p class="mt-0 text-sm">
        {{ t('tools.common-ixps.summary', { ixs: data.ixs.length, facs: data.facilities.length, aIx: data.a.ixCount, bIx: data.b.ixCount, aPolicy: data.a.policy || '—', bPolicy: data.b.policy || '—' }) }}
      </p>
      <h3 class="mb-2 text-sm font-semibold">
        {{ t('tools.common-ixps.ixs') }}
      </h3>
      <ResultTable v-if="ixRows.length" :columns="[{ key: 'name', title: k('ix'), width: 300 }, { key: 'a', title: `AS${data.a.asn}`, width: 150 }, { key: 'b', title: `AS${data.b.asn}`, width: 150 }, { key: 'rs', title: k('rs'), width: 140 }]" :rows="ixRows" row-key="id" filename="common-ixps.json" />
      <p v-else class="text-sm nt-muted">
        {{ t('tools.common-ixps.noIx') }}
      </p>
      <h3 class="mb-2 mt-4 text-sm font-semibold">
        {{ t('tools.common-ixps.facilities') }}
      </h3>
      <ResultTable v-if="facRows.length" :columns="[{ key: 'name', title: k('facility'), width: 340 }, { key: 'where', title: k('where'), width: 200 }]" :rows="facRows" row-key="id" filename="common-facilities.json" />
      <p v-else class="text-sm nt-muted">
        {{ t('tools.common-ixps.noFac') }}
      </p>
    </CCard>
  </template>
</template>
