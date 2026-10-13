<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { parseMac } from '@/utils/mac'
import { fromMac, parsePrefix64, toMac } from './eui64-calculator.service'

const { t } = useI18n()
const mac = useQueryParam({ name: 'mac', defaultValue: '00:1a:2b:3c:4d:5e' })
const prefix = useQueryParam({ name: 'prefix', defaultValue: '2001:db8:1:2::/64' })
const address = useQueryParam({ name: 'addr', defaultValue: '' })

const macValidation = useValidation({ source: mac, rules: [{ validator: (v: string) => parseMac(v) !== undefined, message: t('tools.eui64-calculator.invalidMac') }] })
const prefixValidation = useValidation({ source: prefix, rules: [{ validator: (v: string) => parsePrefix64(v) !== undefined, message: t('tools.eui64-calculator.invalidPrefix') }] })
const addressValidation = useValidation({ source: address, rules: [{ validator: (v: string) => toMac(v) !== undefined, message: t('tools.eui64-calculator.invalidAddress') }] })

const result = computed(() => fromMac(mac.value, prefix.value))
const rows = computed(() => {
  const r = result.value
  if (!r)
    return []
  const k = (key: string) => t(`tools.eui64-calculator.fields.${key}`)
  return [
    { label: k('interfaceId'), value: r.interfaceId, mono: true },
    { label: k('linkLocal'), value: r.linkLocal, mono: true },
    ...(r.slaac ? [{ label: k('slaac'), value: r.slaac, mono: true }] : []),
  ]
})
const recovered = computed(() => toMac(address.value))
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <div class="grid gap-x-4 sm:grid-cols-2">
      <CInputText v-model:value="mac" :label="t('tools.eui64-calculator.mac')" :validation="macValidation" monospace test-id="mac" />
      <CInputText v-model:value="prefix" :label="t('tools.eui64-calculator.prefix')" placeholder="2001:db8:1:2::/64" :validation="prefixValidation" monospace test-id="prefix" />
    </div>
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="eui64.json" />
    </div>
  </CCard>
  <CCard :title="t('tools.eui64-calculator.reverseTitle')">
    <CInputText v-model:value="address" :label="t('tools.eui64-calculator.address')" placeholder="fe80::21a:2bff:fe3c:4d5e" :validation="addressValidation" monospace test-id="address" />
    <CKeyValueList v-if="recovered" :rows="[{ label: t('tools.eui64-calculator.fields.mac'), value: recovered, mono: true }]" data-testid="recovered" />
  </CCard>
</template>
