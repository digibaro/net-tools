<script setup lang="ts">
import type { Converted, Parsed } from './key-converter.service'
import { NAlert, NFormItem, NSwitch } from 'naive-ui'
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { convert, KeyError, parseKey } from './key-converter.service'

const { t } = useI18n()
// Key material is never put in the URL.
const input = ref('')
const showPrivate = ref(false)
const parsed = shallowRef<Parsed>()
const result = shallowRef<Converted>()
const error = ref('')

let seq = 0
watch(input, async (v) => {
  const mine = ++seq
  if (!v.trim()) {
    parsed.value = result.value = undefined
    error.value = ''
    return
  }
  try {
    const p = await parseKey(v)
    const c = await convert(p)
    if (mine !== seq)
      return
    parsed.value = p
    result.value = c
    error.value = ''
  }
  catch (e) {
    if (mine !== seq)
      return
    parsed.value = result.value = undefined
    error.value = e instanceof KeyError ? t(`tools.key-converter.errors.${e.key}`, { detail: e.message }) : t('tools.key-converter.errors.invalid', { detail: String(e) })
  }
})

const validation = computed(() => error.value ? { isValid: false, status: 'error' as const, message: error.value } : undefined)
const infoRows = computed(() => {
  const r = result.value!
  return [
    { label: t('tools.key-converter.rows.type'), value: `${r.type} (${r.bits} bit)`, noCopy: true },
    { label: t('tools.key-converter.rows.source'), value: parsed.value!.source, noCopy: true },
    { label: t('tools.key-converter.rows.kind'), value: r.isPrivate ? t('tools.key-converter.private') : t('tools.key-converter.public'), noCopy: true },
    ...(parsed.value!.comment ? [{ label: t('tools.key-converter.rows.comment'), value: parsed.value!.comment }] : []),
    { label: t('tools.key-converter.rows.sshSha256'), value: r.fingerprints.sshSha256, mono: true },
    { label: t('tools.key-converter.rows.jwkThumbprint'), value: r.fingerprints.jwkThumbprint, mono: true },
    { label: t('tools.key-converter.rows.spkiSha256'), value: r.fingerprints.spkiSha256, mono: true },
  ]
})
const outputs = computed(() => (result.value?.outputs ?? []).filter(o => !o.private || showPrivate.value))
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.key-converter.input')" :placeholder="t('tools.key-converter.placeholder')" :validation="validation" multiline :rows="8" monospace test-id="input" />
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.key-converter.note') }}
    </p>
  </CCard>
  <template v-if="result">
    <CCard :title="t('common.result')" data-testid="result">
      <CKeyValueList :rows="infoRows" />
      <template v-if="result.isPrivate">
        <NAlert type="warning" class="mt-3">
          {{ t('tools.key-converter.privateWarning') }}
        </NAlert>
        <NFormItem :label="t('tools.key-converter.showPrivate')" :show-feedback="false" label-placement="left" class="mt-3">
          <NSwitch v-model:value="showPrivate" :aria-label="t('tools.key-converter.showPrivate')" data-testid="show-private" />
        </NFormItem>
      </template>
    </CCard>
    <CCard v-for="o in outputs" :key="o.id" :title="t(`tools.key-converter.outputs.${o.id}`)" :data-testid="`out-${o.id}`">
      <pre class="m-0 mb-3 overflow-auto whitespace-pre-wrap break-all text-sm font-mono">{{ o.text }}</pre>
      <CopyButton :text="o.text" size="small" />
    </CCard>
  </template>
</template>
