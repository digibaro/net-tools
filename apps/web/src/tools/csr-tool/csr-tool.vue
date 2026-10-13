<script setup lang="ts">
import type { DecodedCsr, KeyType } from './csr-tool.service'
import { NAlert, NButton, NFormItem, NSwitch, NTabPane, NTabs, NTag } from 'naive-ui'
import { computed, reactive, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { downloadText } from '@/utils/download'
import { decodeCsr, generateCsr, KEY_TYPES, sanList, validateSubject } from './csr-tool.service'

const { t } = useI18n()
const tab = useQueryParam({ name: 'tab', defaultValue: 'decode' })

// --- decode ---
const input = ref('')
const decoded = shallowRef<DecodedCsr>()
const invalid = ref(false)
watch(input, async (v) => {
  decoded.value = v.trim() ? await decodeCsr(v) : undefined
  invalid.value = !!v.trim() && !decoded.value
})
const decodedRows = computed(() => {
  const d = decoded.value
  if (!d)
    return []
  const k = (key: string) => t(`tools.csr-tool.fields.${key}`)
  return [
    { label: k('subject'), value: d.subject, mono: true },
    { label: k('sans'), value: d.sans.join(', ') || '—', mono: true },
    { label: k('publicKey'), value: d.publicKey },
    { label: k('signature'), value: d.signatureAlgorithm },
    { label: k('spki'), value: d.spkiSha256, mono: true },
  ]
})

// --- generate ---
const subject = reactive({ cn: '', o: '', ou: '', l: '', st: '', c: '', email: '' })
const sansText = ref('')
const includeCn = ref(true)
const keyType = useQueryParam({ name: 'key', defaultValue: 'ec-p256' })
const sans = computed(() => sanList(sansText.value, subject.cn, includeCn.value))
const subjectError = computed(() => (subject.cn ? validateSubject(subject, sans.value) : undefined))
const generating = ref(false)
const generated = shallowRef<{ privateKeyPem: string, csrPem: string }>()
async function generate() {
  if (!subject.cn || subjectError.value)
    return
  generating.value = true
  try {
    generated.value = await generateCsr(subject, sans.value, keyType.value as KeyType)
  }
  finally {
    generating.value = false
  }
}
const keyOptions = KEY_TYPES.map(k => ({ label: t(`tools.csr-tool.keyTypes.${k}`), value: k }))
const subjectFields = ['cn', 'o', 'ou', 'l', 'st', 'c', 'email'] as const
const baseName = computed(() => (subject.cn || 'request').replace(/^\*\./, 'wildcard.').replace(/[^\w.-]/g, '_'))
</script>

<template>
  <NTabs v-model:value="tab" type="segment" class="mb-4" animated>
    <NTabPane name="decode" :tab="t('tools.csr-tool.decode')">
      <CCard>
        <CInputText v-model:value="input" :label="t('tools.csr-tool.csrInput')" placeholder="-----BEGIN CERTIFICATE REQUEST-----" multiline :rows="8" monospace test-id="csr" :validation="invalid ? { isValid: false, message: t('tools.csr-tool.invalidCsr'), status: 'error' } : undefined" />
      </CCard>
      <CCard v-if="decoded" :title="t('common.result')" data-testid="result">
        <template #header-extra>
          <NTag :type="decoded.signatureValid ? 'success' : 'error'" :bordered="false">
            {{ decoded.signatureValid ? t('tools.csr-tool.sigOk') : t('tools.csr-tool.sigBad') }}
          </NTag>
        </template>
        <CKeyValueList :rows="decodedRows" />
      </CCard>
    </NTabPane>
    <NTabPane name="generate" :tab="t('tools.csr-tool.generate')">
      <CCard>
        <div class="grid gap-x-4 sm:grid-cols-2">
          <CInputText v-for="f in subjectFields" :key="f" v-model:value="subject[f]" :label="t(`tools.csr-tool.subject.${f}`)" :placeholder="t(`tools.csr-tool.placeholders.${f}`)" :test-id="`subject-${f}`" />
        </div>
        <CInputText v-model:value="sansText" :label="t('tools.csr-tool.sans')" placeholder="example.com www.example.com" monospace test-id="sans" />
        <div class="flex flex-wrap items-end gap-x-6">
          <CSelect v-model:value="keyType" :label="t('tools.csr-tool.keyType')" :options="keyOptions" class="min-w-56" />
          <NFormItem :label="t('tools.csr-tool.includeCn')" :show-feedback="false" class="mb-3">
            <NSwitch v-model:value="includeCn" :aria-label="t('tools.csr-tool.includeCn')" />
          </NFormItem>
        </div>
        <p v-if="sans.length" class="mt-0 text-xs nt-muted">
          {{ t('tools.csr-tool.sanPreview', { list: sans.join(', ') }) }}
        </p>
        <NAlert v-if="subjectError" type="error" class="mb-3">
          {{ t(`tools.csr-tool.errors.${subjectError}`) }}
        </NAlert>
        <NButton type="primary" :loading="generating" :disabled="!subject.cn || !!subjectError" data-testid="generate" @click="generate">
          {{ t('tools.csr-tool.generateButton') }}
        </NButton>
      </CCard>
      <template v-if="generated">
        <CCard :title="t('tools.csr-tool.csrTitle')" data-testid="generated-csr">
          <pre class="m-0 overflow-auto text-xs font-mono" tabindex="0">{{ generated.csrPem }}</pre>
          <div class="mt-3 flex flex-wrap gap-2">
            <CopyButton :text="generated.csrPem" :label="t('common.copyText')" />
            <NButton size="small" secondary @click="downloadText(`${baseName}.csr`, generated!.csrPem, 'application/pkcs10')">
              {{ t('tools.csr-tool.download', { file: `${baseName}.csr` }) }}
            </NButton>
          </div>
        </CCard>
        <CCard :title="t('tools.csr-tool.keyTitle')" data-testid="generated-key">
          <NAlert type="warning" class="mb-3">
            {{ t('tools.csr-tool.keyWarning') }}
          </NAlert>
          <pre class="m-0 max-h-60 overflow-auto text-xs font-mono" tabindex="0">{{ generated.privateKeyPem }}</pre>
          <div class="mt-3 flex flex-wrap gap-2">
            <CopyButton :text="generated.privateKeyPem" :label="t('common.copyText')" />
            <NButton size="small" secondary @click="downloadText(`${baseName}.key`, generated!.privateKeyPem, 'application/pkcs8')">
              {{ t('tools.csr-tool.download', { file: `${baseName}.key` }) }}
            </NButton>
          </div>
        </CCard>
      </template>
    </NTabPane>
  </NTabs>
</template>
