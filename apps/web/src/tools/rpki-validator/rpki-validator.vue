<script setup lang="ts">
import type { RpkiResult } from './rpki-validator.service'
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
import { isPrefixOrIp, parseAsn, statusType } from './rpki-validator.service'

const { t, te } = useI18n()
const prefix = useQueryParam({ name: 'prefix', defaultValue: '' })
const asn = useQueryParam({ name: 'asn', defaultValue: '' })
const prefixV = useValidation({ source: prefix, rules: [{ validator: isPrefixOrIp, message: t('bgp.invalidPrefix') }] })
const asnV = useValidation({ source: asn, rules: [{ validator: (v: string) => parseAsn(v) !== undefined, message: t('bgp.invalidAsn') }] })
const canRun = computed(() => !!prefix.value && !!asn.value && prefixV.value.isValid && asnV.value.isValid)
const { data, error, loading, run } = useApi<RpkiResult>('/v1/rpki')
const submit = () => canRun.value && run({ prefix: prefix.value.trim(), asn: asn.value.trim() })
onMounted(() => {
  if (canRun.value)
    void submit()
})
const statusText = (s: string) => (te(`tools.rpki-validator.status.${s}`) ? t(`tools.rpki-validator.status.${s}`) : s)
const roaRows = computed(() => (data.value?.roas ?? []).map(r => ({ prefix: r.prefix, maxLength: r.maxLength, origin: `AS${r.origin}`, validity: statusText(r.validity) })))
const c = (key: string) => t(`bgp.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="prefix" :label="t('bgp.prefix')" placeholder="1.1.1.0/24" :validation="prefixV" monospace test-id="prefix" />
        <CInputText v-model:value="asn" :label="t('tools.rpki-validator.origin')" placeholder="AS13335" :validation="asnV" monospace test-id="asn" />
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!canRun" data-testid="run">
        {{ t('tools.rpki-validator.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="statusType(data.status) === 'default' ? 'info' : statusType(data.status)" class="mb-4" data-testid="result">
      {{ t('tools.rpki-validator.verdict', { prefix: data.prefix, asn: data.asn, status: statusText(data.status) }) }}
    </NAlert>
    <CCard v-if="roaRows.length" :title="t('tools.rpki-validator.roas')">
      <ResultTable :columns="[{ key: 'prefix', title: c('prefix'), mono: true, width: 180 }, { key: 'maxLength', title: c('maxLength'), width: 110 }, { key: 'origin', title: c('origin'), width: 120 }, { key: 'validity', title: c('validity'), width: 240 }]" :rows="roaRows" row-key="prefix" filename="rpki.json" />
    </CCard>
    <p class="text-xs nt-muted">
      {{ t('tools.rpki-validator.source', { validator: data.validator || 'RIPEstat' }) }}
    </p>
  </template>
</template>
