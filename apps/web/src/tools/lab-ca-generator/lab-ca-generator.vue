<script setup lang="ts">
import type { CAError, KeyType, LabResult } from './lab-ca-generator.service'
import { NAlert, NButton, NFormItem, NInput, NInputNumber, NRadioButton, NRadioGroup, NSwitch } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { downloadText } from '@/utils/download'
import { generateLab, KEY_TYPES, sanType } from './lab-ca-generator.service'

const { t } = useI18n()
const caName = useQueryParam({ name: 'ca', defaultValue: 'Homelab Root CA' })
const organization = useQueryParam({ name: 'org', defaultValue: '' })
const keyType = useQueryParam({ name: 'key', defaultValue: 'ecdsa-p256' })
const caDays = useQueryParam({ name: 'caDays', defaultValue: 3650 })
const serverName = useQueryParam({ name: 'cn', defaultValue: 'nas.home.arpa' })
const sansText = useQueryParam({ name: 'san', defaultValue: '' })
const serverDays = useQueryParam({ name: 'days', defaultValue: 825 })
const clientAuth = useQueryParam({ name: 'client', defaultValue: false })
const sans = computed(() => sansText.value.split(/[\s,]+/).filter(Boolean))
const nameValidation = useValidation({ source: serverName, rules: [{ validator: (s: string) => !!sanType(s), message: t('tools.lab-ca-generator.invalidName') }] })
const sanValidation = useValidation({ source: sansText, rules: [{ validator: (s: string) => s.split(/[\s,]+/).filter(Boolean).every(x => !!sanType(x)), message: t('tools.lab-ca-generator.invalidSan') }] })
const caValidation = useValidation({ source: caName, rules: [{ validator: (s: string) => !!s.trim(), message: t('tools.lab-ca-generator.invalidCa') }] })
const keyOptions = computed(() => KEY_TYPES.map(k => ({ label: t(`tools.lab-ca-generator.keys.${k}`), value: k })))
const mode = ref<'new' | 'existing'>('new')
const caCertPem = ref('')
const caKeyPem = ref('')
const caError = ref<CAError>()
const result = shallowRef<LabResult>()
const busy = ref(false)
const valid = computed(() => nameValidation.value.isValid && sanValidation.value.isValid && (mode.value === 'existing' ? !!caCertPem.value.trim() && !!caKeyPem.value.trim() : caValidation.value.isValid))
async function generate() {
  if (!valid.value)
    return
  busy.value = true
  try {
    const r = await generateLab({ caName: caName.value, organization: organization.value, keyType: keyType.value as KeyType, caDays: caDays.value, serverName: serverName.value, sans: sans.value, serverDays: serverDays.value, clientAuth: clientAuth.value }, mode.value === 'existing' ? { certPem: caCertPem.value, keyPem: caKeyPem.value } : undefined)
    caError.value = typeof r === 'string' ? r : undefined
    result.value = typeof r === 'string' ? undefined : r
  }
  finally {
    busy.value = false
  }
}
const files = computed(() => (result.value
  ? [
      { name: 'ca.crt', label: t('tools.lab-ca-generator.files.caCert'), text: result.value.caCert },
      { name: 'ca.key', label: t('tools.lab-ca-generator.files.caKey'), text: result.value.caKey },
      { name: `${serverName.value.replace(/^\*\./, 'wildcard.')}.crt`, label: t('tools.lab-ca-generator.files.serverCert'), text: result.value.serverCert },
      { name: `${serverName.value.replace(/^\*\./, 'wildcard.')}.key`, label: t('tools.lab-ca-generator.files.serverKey'), text: result.value.serverKey },
      { name: 'fullchain.pem', label: t('tools.lab-ca-generator.files.fullchain'), text: result.value.fullchain },
    ]
  : []))
</script>

<template>
  <CCard :title="t('tools.lab-ca-generator.ca')">
    <NRadioGroup v-model:value="mode" name="ca-mode" class="mb-3">
      <NRadioButton value="new" :label="t('tools.lab-ca-generator.newCa')" />
      <NRadioButton value="existing" :label="t('tools.lab-ca-generator.existingCa')" data-testid="existing-ca" />
    </NRadioGroup>
    <template v-if="mode === 'existing'">
      <NFormItem :label="t('tools.lab-ca-generator.caCertPem')" :show-feedback="false" class="mb-3">
        <NInput v-model:value="caCertPem" type="textarea" :rows="4" class="font-mono" placeholder="-----BEGIN CERTIFICATE-----" :input-props="{ 'aria-label': t('tools.lab-ca-generator.caCertPem'), 'spellcheck': false }" />
      </NFormItem>
      <NFormItem :label="t('tools.lab-ca-generator.caKeyPem')" :show-feedback="false" class="mb-3">
        <NInput v-model:value="caKeyPem" type="textarea" :rows="4" class="font-mono" placeholder="-----BEGIN PRIVATE KEY-----" :input-props="{ 'aria-label': t('tools.lab-ca-generator.caKeyPem'), 'spellcheck': false }" />
      </NFormItem>
      <NAlert v-if="caError" type="error">
        {{ t(`tools.lab-ca-generator.caErrors.${caError}`) }}
      </NAlert>
    </template>
    <div v-else class="grid gap-x-4 sm:grid-cols-2">
      <CInputText v-model:value="caName" :label="t('tools.lab-ca-generator.caName')" :validation="caValidation" test-id="ca-name" />
      <CInputText v-model:value="organization" :label="t('tools.lab-ca-generator.organization')" test-id="org" />
      <NFormItem :label="t('tools.lab-ca-generator.caDays')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="caDays" :min="1" :max="7300" :show-button="false" :input-props="{ 'aria-label': t('tools.lab-ca-generator.caDays') }" />
      </NFormItem>
    </div>
  </CCard>
  <CCard :title="t('tools.lab-ca-generator.server')">
    <CSelect v-model:value="keyType" :label="t('tools.lab-ca-generator.keyType')" :options="keyOptions" test-id="key-type" />
    <div class="grid gap-x-4 sm:grid-cols-2">
      <CInputText v-model:value="serverName" :label="t('tools.lab-ca-generator.serverName')" placeholder="nas.home.arpa" :validation="nameValidation" monospace test-id="server-name" />
      <NFormItem :label="t('tools.lab-ca-generator.serverDays')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="serverDays" :min="1" :max="3650" :show-button="false" :input-props="{ 'aria-label': t('tools.lab-ca-generator.serverDays') }" />
      </NFormItem>
    </div>
    <CInputText v-model:value="sansText" :label="t('tools.lab-ca-generator.sans')" placeholder="nas, 192.168.1.10, *.home.arpa" :validation="sanValidation" monospace test-id="sans" />
    <label class="mb-3 flex items-center gap-2 text-sm">
      <NSwitch v-model:value="clientAuth" size="small" />
      {{ t('tools.lab-ca-generator.clientAuth') }}
    </label>
    <NButton type="primary" :loading="busy" :disabled="!valid" data-testid="run" @click="generate">
      {{ t('tools.lab-ca-generator.run') }}
    </NButton>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.lab-ca-generator.note') }}
    </p>
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <NAlert type="warning" class="mb-3">
      {{ t('tools.lab-ca-generator.keep') }}
    </NAlert>
    <div v-for="f in files" :key="f.name" class="mb-3">
      <div class="mb-1 flex flex-wrap items-center gap-2">
        <span class="font-semibold">{{ f.label }}</span>
        <span class="text-xs nt-muted font-mono">{{ f.name }}</span>
        <CopyButton :text="f.text" :label="t('common.copyText')" />
        <NButton size="tiny" @click="downloadText(f.name, f.text, 'application/x-pem-file')">
          {{ t('tools.lab-ca-generator.download') }}
        </NButton>
      </div>
      <pre class="m-0 max-h-32 overflow-auto text-xs font-mono" tabindex="0">{{ f.text }}</pre>
    </div>
    <p class="m-0 text-xs nt-muted">
      {{ t('tools.lab-ca-generator.install') }}
    </p>
  </CCard>
</template>
