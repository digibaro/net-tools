<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { calculateIpv6Subnet } from './ipv6-subnet-calculator.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'ip', defaultValue: '2001:db8:abcd:12::1/48' })
const validation = useValidation({
  source: input,
  rules: [{ validator: (v: string) => !!calculateIpv6Subnet(v), message: t('tools.ipv6-subnet-calculator.invalid') }],
})
const result = computed(() => calculateIpv6Subnet(input.value))

const rows = computed(() => {
  const r = result.value
  if (!r)
    return []
  const k = (key: string) => t(`tools.ipv6-subnet-calculator.fields.${key}`)
  return [
    { label: k('network'), value: r.network, mono: true },
    { label: k('networkExpanded'), value: r.networkExpanded, mono: true },
    { label: k('firstAddress'), value: r.firstAddress, mono: true },
    { label: k('lastAddress'), value: r.lastAddress, mono: true },
    { label: k('lastAddressExpanded'), value: r.lastAddressExpanded, mono: true },
    { label: k('addressExpanded'), value: r.addressExpanded, mono: true },
    { label: k('totalAddresses'), value: r.totalAddresses },
    { label: k('subnets64'), value: r.subnets64 },
    { label: k('subnets48'), value: r.subnets48 },
    { label: k('type'), value: r.type },
    { label: k('reverseZone'), value: r.reverseZone, mono: true },
    { label: k('previous'), value: r.previousNetwork ?? t('common.none'), mono: true },
    { label: k('next'), value: r.nextNetwork ?? t('common.none'), mono: true },
  ]
})
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.ipv6-subnet-calculator.input')" placeholder="2001:db8::/48" :validation="validation" monospace test-id="input" />
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="ipv6-subnet.json" />
    </div>
  </CCard>
</template>
