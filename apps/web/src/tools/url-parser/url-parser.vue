<script setup lang="ts">
import type { Codec } from './url-parser.service'
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { decode, encode, parseUrl } from './url-parser.service'

const { t } = useI18n()
const url = useQueryParam({ name: 'url', defaultValue: 'https://user@example.com:8443/path/to/page?q=network%20tools&lang=en#results' })
const validation = useValidation({ source: url, rules: [{ validator: (v: string) => !!parseUrl(v), message: t('tools.url-parser.invalid') }] })
const parsed = computed(() => parseUrl(url.value))

const rows = computed(() => {
  const p = parsed.value
  if (!p)
    return []
  const k = (key: string) => t(`tools.url-parser.fields.${key}`)
  const v = (s: string) => s || '—'
  return [
    { label: k('protocol'), value: p.protocol, mono: true },
    { label: k('username'), value: v(p.username), mono: true, noCopy: !p.username },
    { label: k('password'), value: p.hasPassword ? '••••••' : '—', noCopy: true },
    { label: k('hostname'), value: p.hostname, mono: true },
    { label: k('port'), value: p.port || t('tools.url-parser.defaultPort', { port: p.effectivePort || '—' }), mono: true },
    { label: k('pathname'), value: p.pathname, mono: true },
    { label: k('search'), value: v(p.search), mono: true, noCopy: !p.search },
    { label: k('hash'), value: v(p.hash), mono: true, noCopy: !p.hash },
    { label: k('origin'), value: p.origin, mono: true },
  ]
})
const json = computed(() => parsed.value && { ...parsed.value, password: undefined })
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')

const codec = ref<Codec>('component')
const plain = ref('name=Ünïcode & spaces/slashes?')
const encoded = computed(() => encode(plain.value, codec.value))
const toDecode = ref('q%3Dnetwork%20tools%26lang%3Den')
const decoded = computed(() => decode(toDecode.value, codec.value))
const decodeValidation = computed(() => decoded.value === undefined ? { isValid: false, status: 'error' as const, message: t('tools.url-parser.malformed') } : undefined)
</script>

<template>
  <CCard>
    <CInputText v-model:value="url" :label="t('tools.url-parser.input')" :validation="validation" monospace test-id="input" />
  </CCard>
  <template v-if="parsed">
    <CCard :title="t('tools.url-parser.components')" data-testid="result">
      <CKeyValueList :rows="rows" />
      <div class="mt-4">
        <ResultActions :text="asText" :json="json" filename="url.json" />
      </div>
    </CCard>
    <CCard v-if="parsed.params.length" :title="t('tools.url-parser.params', { count: parsed.params.length })">
      <ResultTable :columns="[{ key: 'key', title: t('tools.url-parser.key'), mono: true }, { key: 'value', title: t('tools.url-parser.value'), mono: true }]" :rows="parsed.params" filename="query-params.json" />
    </CCard>
  </template>
  <CCard :title="t('tools.url-parser.codecTitle')">
    <NRadioGroup v-model:value="codec" name="codec" class="mb-3">
      <NRadioButton value="component" :label="t('tools.url-parser.codecComponent')" />
      <NRadioButton value="uri" :label="t('tools.url-parser.codecUri')" />
    </NRadioGroup>
    <div class="grid gap-x-4 md:grid-cols-2">
      <div>
        <CInputText v-model:value="plain" :label="t('tools.url-parser.plain')" multiline :rows="3" test-id="plain" />
        <CInputText :value="encoded" :label="t('tools.url-parser.encoded')" multiline :rows="3" readonly monospace test-id="encoded" />
      </div>
      <div>
        <CInputText v-model:value="toDecode" :label="t('tools.url-parser.toDecode')" :validation="decodeValidation" multiline :rows="3" monospace test-id="to-decode" />
        <CInputText :value="decoded ?? ''" :label="t('tools.url-parser.decoded')" multiline :rows="3" readonly test-id="decoded" />
      </div>
    </div>
  </CCard>
</template>
