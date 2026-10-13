<script setup lang="ts">
import type { RecordInput, RecordKind, SpfInput } from './zone-builder.service'
import { NAlert, NButton, NCheckbox, NFormItem, NInputNumber, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { downloadText } from '@/utils/download'
import { splitList } from '@/utils/ip'
import { buildRecord, buildSpf, dateSerial, RECORD_KINDS, zoneHeader } from './zone-builder.service'

const { t } = useI18n()
const origin = useQueryParam({ name: 'origin', defaultValue: 'example.com' })
const ttl = useQueryParam({ name: 'ttl', defaultValue: 3600 })
const primary = useQueryParam({ name: 'ns', defaultValue: 'ns1.example.com.' })
const email = useQueryParam({ name: 'email', defaultValue: 'hostmaster@example.com' })
const records = useQueryParam({ name: 'rr', defaultValue: '' })

const kind = useQueryParam({ name: 'type', defaultValue: 'A' })
const form = reactive({
  name: '@',
  ttl: null as number | null,
  address: '',
  target: '',
  preference: 10,
  text: '',
  service: 'sip',
  protocol: 'tcp',
  priority: 10,
  weight: 5,
  port: 5060,
  critical: false,
  tag: 'issue' as 'issue' | 'issuewild' | 'iodef' | 'issuemail',
  value: '',
  spfMx: true,
  spfA: false,
  spfIncludes: '',
  spfIp4: '',
  spfIp6: '',
  spfAll: '~' as SpfInput['all'],
})

const spfInput = computed<SpfInput>(() => ({ mx: form.spfMx, a: form.spfA, includes: splitList(form.spfIncludes), ip4: splitList(form.spfIp4), ip6: splitList(form.spfIp6), all: form.spfAll }))
const spfPreview = computed(() => buildSpf(spfInput.value))

const input = computed<RecordInput>(() => {
  const common = { name: form.name, ttl: form.ttl ?? undefined }
  switch (kind.value as RecordKind) {
    case 'A':
    case 'AAAA':
      return { ...common, kind: kind.value as 'A' | 'AAAA', address: form.address }
    case 'CNAME':
    case 'NS':
    case 'PTR':
      return { ...common, kind: kind.value as 'CNAME' | 'NS' | 'PTR', target: form.target }
    case 'MX':
      return { ...common, kind: 'MX', preference: form.preference, target: form.target }
    case 'TXT':
      return { ...common, kind: 'TXT', text: form.text }
    case 'SPF':
      return { ...common, kind: 'SPF', spf: spfInput.value }
    case 'SRV':
      return { ...common, kind: 'SRV', service: form.service, protocol: form.protocol, priority: form.priority, weight: form.weight, port: form.port, target: form.target }
    default:
      return { ...common, kind: 'CAA', critical: form.critical, tag: form.tag, value: form.value }
  }
})
const built = computed(() => buildRecord(input.value))
function add() {
  if ('line' in built.value)
    records.value = records.value ? `${records.value.replace(/\n+$/, '')}\n${built.value.line}` : built.value.line
}

const header = computed(() => zoneHeader({ origin: origin.value, ttl: ttl.value, primary: primary.value, email: email.value, serial: dateSerial(), refresh: 7200, retry: 3600, expire: 1209600, minimum: 3600 }))
const zone = computed(() => `${[header.value ?? '', '', records.value.trim()].join('\n').trim()}\n`)
const filename = computed(() => `${origin.value.trim().replace(/\.$/, '') || 'zone'}.zone`)

const typeOptions = RECORD_KINDS.map(k => ({ label: k === 'SPF' ? t('tools.zone-builder.spfType') : k, value: k }))
const tagOptions = ['issue', 'issuewild', 'iodef', 'issuemail'].map(v => ({ label: v, value: v }))
const f = (key: string) => t(`tools.zone-builder.fields.${key}`)
const showTarget = computed(() => ['CNAME', 'NS', 'PTR', 'MX', 'SRV'].includes(kind.value))
</script>

<template>
  <CCard :title="t('tools.zone-builder.zoneTitle')">
    <div class="grid gap-x-4 sm:grid-cols-2">
      <CInputText v-model:value="origin" :label="f('origin')" placeholder="example.com" monospace test-id="origin" />
      <NFormItem :label="f('defaultTtl')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="ttl" :min="0" :show-button="false" :input-props="{ 'aria-label': f('defaultTtl') }" />
      </NFormItem>
      <CInputText v-model:value="primary" :label="f('primary')" placeholder="ns1.example.com." monospace />
      <CInputText v-model:value="email" :label="f('email')" placeholder="hostmaster@example.com" monospace />
    </div>
  </CCard>

  <CCard :title="t('tools.zone-builder.addTitle')">
    <div class="grid gap-x-4 sm:grid-cols-[8rem_1fr_9rem]">
      <CSelect v-model:value="kind" :label="f('type')" :options="typeOptions" test-id="type" />
      <CInputText v-model:value="form.name" :label="kind === 'SRV' ? f('srvBase') : f('name')" placeholder="@, www, _dmarc" monospace test-id="name" />
      <NFormItem :label="f('ttl')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="form.ttl" :min="0" :show-button="false" :placeholder="String(ttl)" clearable :input-props="{ 'aria-label': f('ttl') }" />
      </NFormItem>
    </div>

    <CInputText v-if="kind === 'A' || kind === 'AAAA'" v-model:value="form.address" :label="f('address')" :placeholder="kind === 'A' ? '192.0.2.1' : '2001:db8::1'" monospace test-id="address" />
    <div v-if="kind === 'SRV'" class="grid grid-cols-2 gap-x-4 sm:grid-cols-5">
      <CInputText v-model:value="form.service" :label="f('service')" placeholder="sip" monospace />
      <CInputText v-model:value="form.protocol" :label="f('protocol')" placeholder="tcp" monospace />
      <NFormItem v-for="key in (['priority', 'weight', 'port'] as const)" :key="key" :label="f(key)" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="form[key]" :min="0" :max="65535" :show-button="false" :input-props="{ 'aria-label': f(key) }" />
      </NFormItem>
    </div>
    <div v-if="showTarget" class="grid gap-x-4" :class="kind === 'MX' ? 'sm:grid-cols-[9rem_1fr]' : ''">
      <NFormItem v-if="kind === 'MX'" :label="f('preference')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="form.preference" :min="0" :max="65535" :show-button="false" :input-props="{ 'aria-label': f('preference') }" />
      </NFormItem>
      <CInputText v-model:value="form.target" :label="f('target')" placeholder="mail.example.com." monospace test-id="target" />
    </div>
    <CInputText v-if="kind === 'TXT'" v-model:value="form.text" :label="f('text')" multiline :rows="3" monospace test-id="text" />
    <template v-if="kind === 'CAA'">
      <div class="grid gap-x-4 sm:grid-cols-[10rem_1fr]">
        <CSelect v-model:value="form.tag" :label="f('tag')" :options="tagOptions" />
        <CInputText v-model:value="form.value" :label="f('value')" :placeholder="form.tag === 'iodef' ? 'mailto:security@example.com' : 'letsencrypt.org'" monospace />
      </div>
      <NCheckbox v-model:checked="form.critical" class="mb-3">
        {{ f('critical') }}
      </NCheckbox>
    </template>
    <template v-if="kind === 'SPF'">
      <div class="mb-3 flex flex-wrap gap-4">
        <NCheckbox v-model:checked="form.spfMx">
          {{ f('spfMx') }}
        </NCheckbox>
        <NCheckbox v-model:checked="form.spfA">
          {{ f('spfA') }}
        </NCheckbox>
      </div>
      <CInputText v-model:value="form.spfIncludes" :label="f('spfIncludes')" placeholder="_spf.google.com spf.protection.outlook.com" monospace />
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="form.spfIp4" :label="f('spfIp4')" placeholder="192.0.2.0/24" monospace />
        <CInputText v-model:value="form.spfIp6" :label="f('spfIp6')" placeholder="2001:db8::/32" monospace />
      </div>
      <NFormItem :label="f('spfAll')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="form.spfAll" name="spf-all">
          <NRadioButton value="-" label="-all" />
          <NRadioButton value="~" label="~all" />
          <NRadioButton value="?" label="?all" />
        </NRadioGroup>
      </NFormItem>
      <NAlert v-if="'lookups' in spfPreview && spfPreview.lookups > 10" type="error" class="mb-3">
        {{ t('tools.zone-builder.spfTooMany', { n: spfPreview.lookups }) }}
      </NAlert>
    </template>

    <p class="m-0 mb-3 break-all text-sm font-mono" data-testid="preview">
      <template v-if="'line' in built">
        {{ built.line }}
      </template>
      <span v-else class="nt-muted font-sans">{{ t(`tools.zone-builder.errors.${built.error}`) }}</span>
    </p>
    <NButton type="primary" :disabled="!('line' in built)" data-testid="add" @click="add">
      {{ t('tools.zone-builder.add') }}
    </NButton>
  </CCard>

  <CCard :title="t('tools.zone-builder.recordsTitle')">
    <CInputText v-model:value="records" :label="t('tools.zone-builder.records')" multiline :rows="8" monospace test-id="records" />
  </CCard>
  <CCard :title="t('tools.zone-builder.outputTitle')" data-testid="result">
    <NAlert v-if="!header" type="warning" class="mb-3">
      {{ t('tools.zone-builder.headerInvalid') }}
    </NAlert>
    <pre class="m-0 max-h-120 overflow-auto text-sm font-mono" tabindex="0" data-testid="zone">{{ zone }}</pre>
    <div class="mt-3 flex flex-wrap gap-2">
      <ResultActions :text="() => zone" :json="{ origin, ttl, records: records.split('\n').filter(Boolean) }" filename="zone.json" />
      <NButton size="small" secondary @click="downloadText(filename, zone, 'text/dns')">
        {{ t('tools.zone-builder.download', { file: filename }) }}
      </NButton>
    </div>
  </CCard>
</template>
