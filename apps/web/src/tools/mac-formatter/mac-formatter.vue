<script setup lang="ts">
import type { MacFormat } from './mac-formatter.service'
import { Refresh } from '@vicons/tabler'
import { NButton, NFormItem, NIcon, NInputNumber, NSwitch } from 'naive-ui'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { formatAs, formatList, generateMacs, MAC_FORMATS, parsePrefix } from './mac-formatter.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'mac', defaultValue: '001A.2B3C.4D5E\n00-1a-2b-3c-4d-5f' })
const formatParam = useQueryParam({ name: 'format', defaultValue: 'colon' })
const upper = useQueryParam({ name: 'upper', defaultValue: false })
const format = computed<MacFormat>({
  get: () => (MAC_FORMATS.includes(formatParam.value as MacFormat) ? formatParam.value as MacFormat : 'colon'),
  set: (v) => { formatParam.value = v },
})
const formatOptions = computed(() => MAC_FORMATS.map(f => ({ label: t(`tools.mac-formatter.formats.${f}`), value: f })))

const formatted = computed(() => formatList(input.value, format.value, upper.value))
const validation = computed(() => formatted.value.invalid.length
  ? { isValid: false, status: 'error' as const, message: t('tools.mac-formatter.invalid', { count: formatted.value.invalid.length, entries: formatted.value.invalid.slice(0, 5).join(', ') }) }
  : undefined)

const count = ref(5)
const prefixInput = ref('')
const prefix = computed(() => parsePrefix(prefixInput.value))
const prefixValidation = computed(() => prefix.value ? undefined : { isValid: false, status: 'error' as const, message: t('tools.mac-formatter.invalidPrefix') })
const generated = ref<bigint[]>([])
function generate() {
  if (prefix.value)
    generated.value = generateMacs(Math.min(Math.max(count.value, 1), 1000), prefix.value)
}
generate()
const generatedText = computed(() => generated.value.map(m => formatAs(m, format.value, upper.value)))
</script>

<template>
  <CCard>
    <div class="flex flex-wrap items-end gap-x-6">
      <div class="min-w-48">
        <CSelect v-model:value="format" :label="t('tools.mac-formatter.format')" :options="formatOptions" test-id="format" />
      </div>
      <NFormItem :label="t('tools.mac-formatter.uppercase')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="upper" :aria-label="t('tools.mac-formatter.uppercase')" />
      </NFormItem>
    </div>
  </CCard>
  <CCard :title="t('tools.mac-formatter.formatTitle')">
    <CInputText v-model:value="input" :label="t('tools.mac-formatter.input')" :validation="validation" multiline :rows="4" monospace test-id="input" />
    <template v-if="formatted.output.length">
      <pre class="m-0 mb-3 overflow-auto text-sm font-mono" data-testid="formatted">{{ formatted.output.join('\n') }}</pre>
      <ResultActions :text="formatted.output.join('\n')" :json="formatted.output" filename="macs.json" />
    </template>
  </CCard>
  <CCard :title="t('tools.mac-formatter.generateTitle')">
    <div class="grid gap-x-4 sm:grid-cols-[10rem_1fr]">
      <NFormItem :label="t('tools.mac-formatter.count')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="count" :min="1" :max="1000" :show-button="false" :input-props="{ 'aria-label': t('tools.mac-formatter.count') }" />
      </NFormItem>
      <CInputText v-model:value="prefixInput" :label="t('tools.mac-formatter.prefix')" placeholder="00:0c:29" :validation="prefixValidation" monospace />
    </div>
    <p class="mt-0 text-sm nt-muted">
      {{ t('tools.mac-formatter.generateHint') }}
    </p>
    <NButton type="primary" class="mb-3" :disabled="!prefix" data-testid="generate" @click="generate">
      <template #icon>
        <NIcon aria-hidden="true" :component="Refresh" />
      </template>
      {{ t('common.generate') }}
    </NButton>
    <pre class="m-0 mb-3 overflow-auto text-sm font-mono" data-testid="generated">{{ generatedText.join('\n') }}</pre>
    <ResultActions :text="generatedText.join('\n')" :json="generatedText" filename="random-macs.json" />
  </CCard>
</template>
