<script setup lang="ts">
import type { Dns64Result } from './nat64-dns64-check.service'
import { NAlert, NButton } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isResolver, verdict } from './nat64-dns64-check.service'

const { t } = useI18n()
const resolver = useQueryParam({ name: 'resolver', defaultValue: '' })
const validation = useValidation({ source: resolver, rules: [{ validator: isResolver, message: t('tools.nat64-dns64-check.invalidResolver') }] })
const { data, error, loading, run } = useApi<Dns64Result>('/v1/dns/dns64')
const submit = () => validation.value.isValid && run({ resolver: resolver.value.trim() || 'system' })
const v = computed(() => (data.value ? verdict(data.value) : undefined))
const alertType = { wellKnown: 'success', networkSpecific: 'success', none: 'info' } as const
const k = (key: string) => t(`tools.nat64-dns64-check.fields.${key}`)
const rows = computed(() => data.value
  ? [
      { label: k('resolver'), value: data.value.resolver, mono: true },
      { label: k('prefixes'), value: data.value.prefixes.map(p => p.prefix).join(', ') || t('common.none'), mono: true },
      { label: k('synthesized'), value: data.value.synthesized.join(', ') || t('common.none'), mono: true },
      { label: k('ipv4'), value: data.value.ipv4.join(', ') || t('common.none'), mono: true },
      { label: k('rcode'), value: data.value.rcode, noCopy: true },
    ]
  : [])
const asText = () => [v.value ? t(`tools.nat64-dns64-check.verdict.${v.value}`, { prefix: data.value?.prefixes[0]?.prefix }) : '', ...rows.value.map(r => `${r.label}: ${r.value}`)].join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="resolver" :label="t('tools.nat64-dns64-check.resolver')" :placeholder="t('tools.nat64-dns64-check.resolverPlaceholder')" :validation="validation" monospace test-id="resolver" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!validation.isValid" data-testid="run">
        {{ t('tools.nat64-dns64-check.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.nat64-dns64-check.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data && v" :title="t('common.result')" data-testid="result">
    <NAlert :type="alertType[v]" class="mb-3" data-testid="verdict">
      {{ t(`tools.nat64-dns64-check.verdict.${v}`, { prefix: data.prefixes[0]?.prefix }) }}
    </NAlert>
    <CKeyValueList :rows="rows" />
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="dns64.json" />
    </div>
  </CCard>
</template>
