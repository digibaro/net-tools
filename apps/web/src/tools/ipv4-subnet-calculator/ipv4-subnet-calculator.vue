<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { calculateIpv4Subnet, parseIpv4Input } from './ipv4-subnet-calculator.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'ip', defaultValue: '192.168.1.10/24' })
const validation = useValidation({
  source: input,
  rules: [{ validator: (v: string) => !!parseIpv4Input(v), message: t('tools.ipv4-subnet-calculator.invalid') }],
})

const result = computed(() => {
  const parsed = parseIpv4Input(input.value)
  return parsed ? calculateIpv4Subnet(parsed.address, parsed.prefix) : undefined
})

const rows = computed(() => {
  const r = result.value
  if (!r)
    return []
  const k = (key: string) => t(`tools.ipv4-subnet-calculator.fields.${key}`)
  return [
    { label: k('network'), value: `${r.network}/${r.prefix}`, mono: true },
    { label: k('netmask'), value: r.netmask, mono: true },
    { label: k('wildcard'), value: r.wildcard, mono: true },
    { label: k('broadcast'), value: r.broadcast ?? t('common.none'), mono: true },
    { label: k('hostRange'), value: `${r.firstHost} – ${r.lastHost}`, mono: true },
    { label: k('usableHosts'), value: r.usableHosts },
    { label: k('totalAddresses'), value: r.totalAddresses },
    { label: k('class'), value: r.addressClass, noCopy: true },
    { label: k('type'), value: r.type },
    { label: k('binaryAddress'), value: r.binaryAddress, mono: true },
    { label: k('binaryNetmask'), value: r.binaryNetmask, mono: true },
    { label: k('hex'), value: r.hexAddress, mono: true },
    { label: k('integer'), value: r.integerAddress, mono: true },
    { label: k('previous'), value: r.previousNetwork ?? t('common.none'), mono: true },
    { label: k('next'), value: r.nextNetwork ?? t('common.none'), mono: true },
  ]
})

const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText
      v-model:value="input"
      :label="t('tools.ipv4-subnet-calculator.input')"
      placeholder="192.168.1.10/24"
      :validation="validation"
      monospace
      test-id="input"
    />
    <p class="m-0 text-sm nt-muted">
      {{ t('tools.ipv4-subnet-calculator.hint') }}
    </p>
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="ipv4-subnet.json" />
    </div>
  </CCard>
</template>
