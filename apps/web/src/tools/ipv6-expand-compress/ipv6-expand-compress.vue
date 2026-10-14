<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { ipv6Forms } from './ipv6-expand-compress.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'ip', defaultValue: '2001:0db8:0000:0000:0000:ff00:0042:8329' })
const validation = useValidation({
  source: input,
  rules: [{ validator: (v: string) => !!ipv6Forms(v), message: t('tools.ipv6-expand-compress.invalid') }],
})
const result = computed(() => ipv6Forms(input.value))
const rows = computed(() => Object.entries(result.value ?? {}).map(([key, value]) => ({
  label: t(`tools.ipv6-expand-compress.fields.${key}`),
  value,
  mono: key !== 'type',
})))
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.ipv6-expand-compress.input')" placeholder="2001:db8::1" :validation="validation" monospace test-id="input" />
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="ipv6-forms.json" />
    </div>
  </CCard>
</template>
