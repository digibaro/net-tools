<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { aggregate } from './cidr-aggregator.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'list', defaultValue: '10.0.0.0/24\n10.0.1.0/24\n10.0.2.0/25\n10.0.2.128/25\n2001:db8::/33\n2001:db8:8000::/33' })
const result = computed(() => aggregate(input.value))
const validation = computed(() => result.value.invalid.length
  ? { isValid: false, status: 'error' as const, message: t('tools.cidr-aggregator.invalid', { entries: result.value.invalid.slice(0, 5).join(', '), count: result.value.invalid.length }) }
  : undefined)
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.cidr-aggregator.input')" :validation="validation" multiline :rows="8" monospace test-id="input" />
  </CCard>
  <CCard v-if="result.output.length" :title="t('tools.cidr-aggregator.output')" data-testid="result">
    <p class="mt-0 text-sm nt-muted">
      {{ t('tools.cidr-aggregator.summary', { input: result.inputCount, output: result.output.length }) }}
    </p>
    <pre class="m-0 mb-4 overflow-auto text-sm font-mono" data-testid="output">{{ result.output.join('\n') }}</pre>
    <ResultActions :text="result.output.join('\n')" :json="result.output" filename="aggregated-prefixes.json" />
  </CCard>
</template>
