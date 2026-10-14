<script setup lang="ts">
import { NAlert } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { parseIPv4 } from '@/utils/ip'
import { analyzeMask } from './wildcard-mask-calculator.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'mask', defaultValue: '0.0.0.255' })
const network = useQueryParam({ name: 'net', defaultValue: '' })
const validation = useValidation({ source: input, rules: [{ validator: (v: string) => !!analyzeMask(v), message: t('tools.wildcard-mask-calculator.invalid') }] })
const netValidation = useValidation({ source: network, rules: [{ validator: (v: string) => parseIPv4(v) !== undefined, message: t('tools.wildcard-mask-calculator.invalidNetwork') }] })
const result = computed(() => analyzeMask(input.value, network.value))

const rows = computed(() => {
  const r = result.value
  if (!r)
    return []
  const k = (key: string) => t(`tools.wildcard-mask-calculator.fields.${key}`)
  return [
    { label: k('kind'), value: t(`tools.wildcard-mask-calculator.kinds.${r.kind}`), noCopy: true },
    { label: k('prefix'), value: r.prefix === null ? t('tools.wildcard-mask-calculator.noPrefix') : `/${r.prefix}`, mono: true },
    { label: k('netmask'), value: r.netmask, mono: true },
    { label: k('wildcard'), value: r.wildcard, mono: true },
    { label: k('binaryWildcard'), value: r.binaryWildcard, mono: true },
    { label: k('matching'), value: r.matchingAddresses },
    ...(r.aclExample ? [{ label: k('acl'), value: r.aclExample, mono: true }] : []),
    ...(r.ospfExample ? [{ label: k('ospf'), value: r.ospfExample, mono: true }] : []),
  ]
})
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <div class="grid gap-x-4 sm:grid-cols-2">
      <CInputText v-model:value="input" :label="t('tools.wildcard-mask-calculator.input')" placeholder="/24, 255.255.255.0 or 0.0.0.255" :validation="validation" monospace test-id="input" />
      <CInputText v-model:value="network" :label="t('tools.wildcard-mask-calculator.network')" placeholder="10.0.0.0" :validation="netValidation" monospace test-id="network" />
    </div>
  </CCard>
  <NAlert v-if="result && !result.contiguous" type="info" class="mb-4" data-testid="noncontiguous">
    {{ t('tools.wildcard-mask-calculator.nonContiguous') }}
  </NAlert>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="wildcard-mask.json" />
    </div>
  </CCard>
</template>
