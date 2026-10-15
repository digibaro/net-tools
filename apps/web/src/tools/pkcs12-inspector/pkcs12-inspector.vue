<script setup lang="ts">
import type { P12Error, P12Result } from './pkcs12-inspector.service'
import { Upload } from '@vicons/tabler'
import { NAlert, NButton, NIcon, NInput } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { downloadText } from '@/utils/download'
import { readPkcs12 } from './pkcs12-inspector.service'

const { t } = useI18n()
const fileInput = ref<HTMLInputElement>()
const fileName = ref('')
const bytes = shallowRef<Uint8Array>()
const password = ref('')
const result = shallowRef<P12Result>()
const failure = ref<P12Error>()
const busy = ref(false)
const showKey = ref(false)
async function onFile(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file)
    return
  fileName.value = file.name
  bytes.value = new Uint8Array(await file.arrayBuffer())
  result.value = undefined
  failure.value = undefined
}
async function open() {
  if (!bytes.value)
    return
  busy.value = true
  const r = await readPkcs12(bytes.value, password.value)
  busy.value = false
  if (typeof r === 'string') {
    failure.value = r
    result.value = undefined
  }
  else {
    failure.value = undefined
    result.value = r
  }
}
const k = (key: string) => t(`tools.pkcs12-inspector.fields.${key}`)
function certRows(i: number) {
  const c = result.value!.certs[i]!
  return [
    { label: k('subject'), value: c.subject, mono: true },
    { label: k('issuer'), value: c.selfSigned ? `${c.issuer} (${t('tools.pkcs12-inspector.selfSigned')})` : c.issuer, mono: true },
    { label: k('valid'), value: `${c.notBefore.slice(0, 10)} – ${c.notAfter.slice(0, 10)}`, noCopy: true },
    { label: 'SHA-256', value: c.sha256, mono: true },
    ...(c.friendlyName ? [{ label: k('friendlyName'), value: c.friendlyName, mono: false }] : []),
    ...(c.localKeyId ? [{ label: k('localKeyId'), value: c.localKeyId, mono: true }] : []),
  ]
}
const allCerts = computed(() => (result.value?.certs ?? []).map(c => c.pem).join(''))
</script>

<template>
  <CCard>
    <div class="mb-3 flex flex-wrap items-center gap-2">
      <input ref="fileInput" type="file" accept=".p12,.pfx,application/x-pkcs12" class="hidden" :aria-label="t('tools.pkcs12-inspector.upload')" data-testid="file" @change="onFile">
      <NButton @click="fileInput?.click()">
        <template #icon>
          <NIcon aria-hidden="true" :component="Upload" />
        </template>
        {{ t('tools.pkcs12-inspector.upload') }}
      </NButton>
      <span class="text-sm font-mono">{{ fileName }}</span>
    </div>
    <form class="contents" @submit.prevent="open">
      <NInput v-model:value="password" type="password" show-password-on="click" :placeholder="t('tools.pkcs12-inspector.password')" :input-props="{ 'aria-label': t('tools.pkcs12-inspector.password'), 'autocomplete': 'off' }" class="mb-3" data-testid="password" />
      <NButton type="primary" attr-type="submit" :loading="busy" :disabled="!bytes" data-testid="run">
        {{ t('tools.pkcs12-inspector.run') }}
      </NButton>
    </form>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.pkcs12-inspector.note') }}
    </p>
  </CCard>
  <NAlert v-if="failure" type="error" class="mb-4" data-testid="failure">
    {{ t(`tools.pkcs12-inspector.errors.${failure}`) }}
  </NAlert>
  <template v-if="result">
    <CCard :title="t('tools.pkcs12-inspector.protection')" data-testid="result">
      <NAlert v-if="result.legacy" type="warning" class="mb-3">
        {{ t('tools.pkcs12-inspector.legacy') }}
      </NAlert>
      <CKeyValueList :rows="[{ label: 'MAC', value: result.mac, noCopy: true }, ...result.encryption.map((e, i) => ({ label: `${k('encryption')} ${i + 1}`, value: e, noCopy: true }))]" />
    </CCard>
    <CCard v-for="(c, i) in result.certs" :key="c.sha256" :title="t('tools.pkcs12-inspector.certificate', { n: i + 1 })">
      <CKeyValueList :rows="certRows(i)" />
    </CCard>
    <CCard v-for="(key, i) in result.keys" :key="i" :title="t('tools.pkcs12-inspector.key', { n: i + 1 })">
      <CKeyValueList :rows="[{ label: k('type'), value: key.description, noCopy: true }, { label: k('matches'), value: key.matchesCert >= 0 ? t('tools.pkcs12-inspector.matches', { n: key.matchesCert + 1 }) : t('tools.pkcs12-inspector.noMatch'), noCopy: true }, ...(key.friendlyName ? [{ label: k('friendlyName'), value: key.friendlyName, noCopy: true }] : [])]" />
      <NButton size="small" class="mt-2" @click="showKey = !showKey">
        {{ showKey ? t('tools.pkcs12-inspector.hideKey') : t('tools.pkcs12-inspector.showKey') }}
      </NButton>
      <pre v-if="showKey" class="mt-2 overflow-auto text-xs font-mono" tabindex="0">{{ key.pem }}</pre>
    </CCard>
    <div class="mb-4 flex flex-wrap gap-2">
      <CopyButton :text="allCerts" :label="t('tools.pkcs12-inspector.copyCerts')" />
      <NButton size="small" @click="downloadText('certificates.pem', allCerts, 'application/x-pem-file')">
        {{ t('tools.pkcs12-inspector.downloadCerts') }}
      </NButton>
    </div>
  </template>
</template>
