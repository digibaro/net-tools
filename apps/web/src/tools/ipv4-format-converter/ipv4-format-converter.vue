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
import { formats, parseAnyIPv4 } from './ipv4-format-converter.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'ip', defaultValue: '192.168.1.10' })
const validation = useValidation({
  source: input,
  rules: [{ validator: (v: string) => !!parseAnyIPv4(v), message: t('tools.ipv4-format-converter.invalid') }],
})
const parsed = computed(() => parseAnyIPv4(input.value))
const result = computed(() => (parsed.value ? formats(parsed.value.value) : undefined))
const rows = computed(() => Object.entries(result.value ?? {}).map(([key, value]) => ({
  label: t(`tools.ipv4-format-converter.fields.${key}`),
  value,
  mono: true,
})))
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.ipv4-format-converter.input')" placeholder="127.0.0.1, 2130706433, 0x7f.1, 0177.0.0.1…" :validation="validation" monospace test-id="input" />
  </CCard>
  <NAlert v-if="parsed?.form === 'inet_aton'" type="info" class="mb-4" data-testid="lenient-note">
    {{ t('tools.ipv4-format-converter.lenient', { ip: result?.dotted }) }}
  </NAlert>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="ipv4-formats.json" />
    </div>
  </CCard>
</template>
