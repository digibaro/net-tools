<script setup lang="ts">
import { NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { clientHints, parseUserAgent } from './user-agent-parser.service'

const { t } = useI18n()
const ua = useQueryParam({ name: 'ua', defaultValue: '' })
function useMine() {
  ua.value = navigator.userAgent
}
onMounted(() => {
  if (!ua.value)
    useMine()
})
const isMine = computed(() => ua.value === navigator.userAgent)
const result = computed(() => parseUserAgent(ua.value))
const hints = clientHints()

const rows = computed(() => {
  const r = result.value
  if (!r)
    return []
  const k = (key: string) => t(`tools.user-agent-parser.fields.${key}`)
  const v = (s: string) => s || t('tools.user-agent-parser.unknown')
  return [
    { label: k('browser'), value: v(r.browser) },
    { label: k('engine'), value: v(r.engine) },
    { label: k('os'), value: v(r.os) },
    { label: k('device'), value: v(r.device) },
    { label: k('deviceType'), value: t(`tools.user-agent-parser.types.${r.deviceType}`, r.deviceType), noCopy: true },
    { label: k('cpu'), value: v(r.cpu) },
  ]
})
const hintRows = computed(() => hints
  ? [
      { label: t('tools.user-agent-parser.fields.brands'), value: hints.brands },
      { label: t('tools.user-agent-parser.fields.platform'), value: hints.platform },
      { label: t('tools.user-agent-parser.fields.mobile'), value: t(hints.mobile ? 'common.yes' : 'common.no'), noCopy: true },
    ]
  : [])
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText v-model:value="ua" :label="t('tools.user-agent-parser.input')" multiline :rows="3" monospace test-id="input" />
    <NButton size="small" :disabled="isMine" @click="useMine">
      {{ t('tools.user-agent-parser.useMine') }}
    </NButton>
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="user-agent.json" />
    </div>
  </CCard>
  <CCard v-if="isMine && hintRows.length" :title="t('tools.user-agent-parser.hintsTitle')">
    <p class="mt-0 text-sm nt-muted">
      {{ t('tools.user-agent-parser.hintsNote') }}
    </p>
    <CKeyValueList :rows="hintRows" />
  </CCard>
</template>
