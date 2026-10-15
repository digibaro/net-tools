<script setup lang="ts">
import type { Start } from './packet-decoder.service'
import { NAlert, NFormItem, NInput } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { decode, parseHex } from './packet-decoder.service'

const { t } = useI18n()
const SAMPLE = '0011223344550a0b0c0d0e0f080045000039a1b20000401106e6c0a80164080808080d2c0035002500001a2b01000001000000000000076578616d706c6503636f6d0000010001'
const hex = useQueryParam({ name: 'hex', defaultValue: SAMPLE })
const start = useQueryParam({ name: 'start', defaultValue: 'auto' })
const bytes = computed(() => parseHex(hex.value))
const result = computed(() => (bytes.value?.length ? decode(bytes.value, start.value as Start) : undefined))
const startOptions = computed(() => (['auto', 'ethernet', 'ipv4', 'ipv6', 'sll', 'sll2'] as const).map(v => ({ label: t(`tools.packet-decoder.starts.${v}`), value: v })))
const asText = () => (result.value?.layers ?? []).map(l => [`${l.name}: ${l.summary}`, ...l.fields.map(f => `  ${f.name}: ${f.value}`)].join('\n')).join('\n')
</script>

<template>
  <CCard>
    <NFormItem :label="t('tools.packet-decoder.hex')" :show-feedback="false" class="mb-1">
      <NInput v-model:value="hex" type="textarea" :autosize="{ minRows: 4, maxRows: 14 }" class="font-mono" :input-props="{ 'aria-label': t('tools.packet-decoder.hex'), 'spellcheck': false }" data-testid="hex" />
    </NFormItem>
    <p class="mb-3 mt-0 text-xs nt-muted">
      {{ bytes ? t('tools.packet-decoder.bytes', { n: bytes.length }) : t('tools.packet-decoder.invalid') }}
    </p>
    <CSelect v-model:value="start" :label="t('tools.packet-decoder.start')" :options="startOptions" test-id="start" />
  </CCard>
  <template v-if="result">
    <NAlert v-if="result.error" type="warning" class="mb-4">
      {{ t('tools.packet-decoder.truncated', { detail: result.error }) }}
    </NAlert>
    <CCard v-for="(l, i) in result.layers" :key="i" :title="`${l.name} — ${l.summary}`" data-testid="layer">
      <p class="mt-0 text-xs nt-muted">
        {{ t('tools.packet-decoder.offset', { offset: l.offset, length: l.length }) }}
      </p>
      <CKeyValueList v-if="l.fields.length" :rows="l.fields.map(f => ({ label: f.name, value: f.value, mono: true }))" />
    </CCard>
    <div v-if="result.layers.length" class="mb-4">
      <ResultActions :text="asText" :json="result" filename="packet.json" />
    </div>
  </template>
</template>
