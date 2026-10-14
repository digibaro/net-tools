<script setup lang="ts">
import { NAlert } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { parseIP } from '@/utils/ip'
import { embed, extract, extractCustom } from './ipv4-in-ipv6.service'

const { t } = useI18n()
const address = useQueryParam({ name: 'ip', defaultValue: '192.0.2.33' })
const prefix = useQueryParam({ name: 'prefix', defaultValue: '64:ff9b::/96' })
const parsed = computed(() => parseIP(address.value.trim()))
const validation = useValidation({ source: address, rules: [{ validator: (v: string) => !!parseIP(v.trim()), message: t('tools.ipv4-in-ipv6.invalid') }] })

const embedded = computed(() => (parsed.value?.version === 4 ? embed(address.value, prefix.value) : undefined))
const prefixValidation = useValidation({ source: prefix, rules: [{ validator: (v: string) => !embed('0.0.0.0', v)?.prefixError, message: t('tools.ipv4-in-ipv6.invalidPrefix') }] })
const k = (key: string) => t(`tools.ipv4-in-ipv6.kinds.${key}`)
const embedRows = computed(() => (embedded.value?.forms ?? []).map(f => ({ kind: k(f.id) + (f.note && f.note !== 'deprecated' ? ` (${f.note})` : ''), address: f.address, note: f.note === 'deprecated' ? t('tools.ipv4-in-ipv6.deprecated') : '' })))

const extracted = computed(() => {
  if (parsed.value?.version !== 6)
    return undefined
  const list = extract(address.value) ?? []
  const custom = extractCustom(address.value, prefix.value)
  if (custom && !list.some(e => e.ipv4 === custom))
    list.push({ kind: 'nat64Custom', ipv4: custom })
  return list.map(e => ({ kind: k(e.kind), ipv4: e.ipv4, detail: e.detail ?? '' }))
})
const c = (key: string) => t(`tools.ipv4-in-ipv6.columns.${key}`)
</script>

<template>
  <CCard>
    <CInputText v-model:value="address" :label="t('tools.ipv4-in-ipv6.input')" placeholder="192.0.2.33 or 64:ff9b::c000:221" :validation="validation" monospace test-id="input" />
    <CInputText v-model:value="prefix" :label="t('tools.ipv4-in-ipv6.prefix')" placeholder="64:ff9b::/96" :validation="prefixValidation" monospace test-id="prefix" />
  </CCard>
  <CCard v-if="embedded" :title="t('tools.ipv4-in-ipv6.embedTitle')" data-testid="result">
    <ResultTable :columns="[{ key: 'kind', title: c('kind'), width: 260 }, { key: 'address', title: c('ipv6'), mono: true, width: 260 }, { key: 'note', title: '', width: 120 }]" :rows="embedRows" row-key="address" filename="ipv4-in-ipv6.json" />
  </CCard>
  <CCard v-else-if="extracted" :title="t('tools.ipv4-in-ipv6.extractTitle')" data-testid="result">
    <ResultTable v-if="extracted.length" :columns="[{ key: 'kind', title: c('kind'), width: 260 }, { key: 'ipv4', title: c('ipv4'), mono: true, width: 160 }, { key: 'detail', title: c('detail'), width: 240 }]" :rows="extracted" row-key="kind" filename="ipv4-from-ipv6.json" />
    <NAlert v-else type="info" data-testid="none">
      {{ t('tools.ipv4-in-ipv6.none') }}
    </NAlert>
  </CCard>
</template>
