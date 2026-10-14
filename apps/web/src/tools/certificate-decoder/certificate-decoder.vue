<script setup lang="ts">
import type { DecodedCert } from './certificate-decoder.service'
import { Upload } from '@vicons/tabler'
import { watchDebounced } from '@vueuse/core'
import { NButton, NIcon, NTag } from 'naive-ui'
import { ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { decodeAll } from './certificate-decoder.service'

// Certificates are not mirrored to the URL (too large); everything stays in the browser.
const { t } = useI18n()
const input = ref('')
const certs = shallowRef<DecodedCert[]>()
const invalid = ref(false)
const fileInput = ref<HTMLInputElement>()

watchDebounced(input, async (value) => {
  if (!value.trim()) {
    certs.value = undefined
    invalid.value = false
    return
  }
  certs.value = await decodeAll(value)
  invalid.value = !certs.value
}, { debounce: 200, immediate: true })

async function onFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file)
    return
  const bytes = new Uint8Array(await file.arrayBuffer())
  const text = new TextDecoder().decode(bytes)
  input.value = text.includes('-----BEGIN') ? text : btoa(String.fromCharCode(...bytes))
}

const validityType = { 'valid': 'success', 'expired': 'error', 'not-yet-valid': 'warning' } as const

function rows(c: DecodedCert) {
  const k = (key: string) => t(`tools.certificate-decoder.fields.${key}`)
  const list = (a: string[]) => a.join(', ')
  return [
    { label: k('subject'), value: c.subject, mono: true },
    { label: k('issuer'), value: c.issuer, mono: true },
    { label: k('san'), value: list(c.subjectAltNames), mono: true },
    { label: k('notBefore'), value: c.notBefore },
    { label: k('notAfter'), value: c.notAfter },
    { label: k('serial'), value: c.serialNumber, mono: true },
    { label: k('publicKey'), value: c.publicKey },
    { label: k('signature'), value: c.signatureAlgorithm },
    { label: k('keyUsage'), value: list(c.keyUsage) },
    { label: k('eku'), value: list(c.extendedKeyUsage) },
    { label: k('basicConstraints'), value: c.basicConstraints },
    { label: k('policies'), value: list(c.policies) },
    { label: k('ocsp'), value: list(c.ocsp), mono: true },
    { label: k('caIssuers'), value: list(c.caIssuers), mono: true },
    { label: k('crl'), value: list(c.crlDistributionPoints), mono: true },
    { label: k('ski'), value: c.subjectKeyId, mono: true },
    { label: k('aki'), value: c.authorityKeyId, mono: true },
    { label: k('sha256'), value: c.sha256, mono: true },
    { label: k('sha1'), value: c.sha1, mono: true },
  ].filter(r => r.value)
}
const asText = (c: DecodedCert) => () => rows(c).map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText
      v-model:value="input"
      :label="t('tools.certificate-decoder.input')"
      placeholder="-----BEGIN CERTIFICATE-----"
      :validation="invalid ? { isValid: false, status: 'error', message: t('tools.certificate-decoder.invalid') } : undefined"
      multiline :rows="8" monospace test-id="input"
    />
    <input ref="fileInput" type="file" accept=".pem,.crt,.cer,.der" class="hidden" :aria-label="t('tools.certificate-decoder.upload')" data-testid="file" @change="onFile">
    <NButton size="small" @click="fileInput?.click()">
      <template #icon>
        <NIcon aria-hidden="true" :component="Upload" />
      </template>
      {{ t('tools.certificate-decoder.upload') }}
    </NButton>
    <p class="mb-0 text-sm nt-muted">
      {{ t('tools.certificate-decoder.hint') }}
    </p>
  </CCard>
  <CCard v-for="(c, i) in certs" :key="c.sha256 + i" :title="certs!.length > 1 ? t('tools.certificate-decoder.nth', { n: i + 1 }) : t('common.result')" data-testid="result">
    <template #header-extra>
      <div class="flex flex-wrap gap-1">
        <NTag :type="validityType[c.validity]" size="small" :bordered="false" data-testid="validity">
          {{ t(`tools.certificate-decoder.validity.${c.validity}`, { days: c.daysRemaining }) }}
        </NTag>
        <NTag v-if="c.selfSigned" size="small" :bordered="false">
          {{ t('tools.certificate-decoder.selfSigned') }}
        </NTag>
      </div>
    </template>
    <CKeyValueList :rows="rows(c)" />
    <div class="mt-4">
      <ResultActions :text="asText(c)" :json="c" filename="certificate.json" />
    </div>
  </CCard>
</template>
