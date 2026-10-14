<script setup lang="ts">
import type { Comparison, ItemKind } from './cert-key-matcher.service'
import { NAlert, NTag } from 'naive-ui'
import { computed, reactive, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { compare } from './cert-key-matcher.service'

const { t } = useI18n()
// Private keys must never end up in the URL, history or server logs: plain local state only.
const inputs = reactive<Record<ItemKind, string>>({ certificate: '', key: '', csr: '' })
const result = shallowRef<Comparison>()
let seq = 0
watch(inputs, async () => {
  const mine = ++seq
  const r = await compare({ ...inputs })
  if (mine === seq)
    result.value = r
}, { immediate: true })
const kinds = ['certificate', 'key', 'csr'] as const
const placeholders = { certificate: '-----BEGIN CERTIFICATE-----', key: '-----BEGIN PRIVATE KEY-----', csr: '-----BEGIN CERTIFICATE REQUEST-----' }
const readable = computed(() => result.value?.items.filter(i => i.spki) ?? [])
const reference = computed(() => readable.value[0]?.spki)
</script>

<template>
  <CCard>
    <div class="grid gap-x-4 lg:grid-cols-3">
      <CInputText
        v-for="k in kinds" :key="k" v-model:value="inputs[k]" :label="t(`tools.cert-key-matcher.inputs.${k}`)" :placeholder="placeholders[k]" multiline :rows="7" monospace :test-id="k"
        :validation="result?.items.find(i => i.kind === k)?.error ? { isValid: false, message: t(`tools.cert-key-matcher.errors.${result!.items.find(i => i.kind === k)!.error}`), status: 'error' } : undefined"
      />
    </div>
    <p class="m-0 text-xs nt-muted">
      {{ t('tools.cert-key-matcher.privacy') }}
    </p>
  </CCard>
  <template v-if="result">
    <NAlert v-if="result.match !== undefined" :type="result.match ? 'success' : 'error'" class="mb-4" data-testid="verdict">
      {{ result.match ? t('tools.cert-key-matcher.match') : t('tools.cert-key-matcher.mismatch') }}
    </NAlert>
    <CCard v-if="readable.length" :title="t('tools.cert-key-matcher.keys')" data-testid="result">
      <div v-for="i in readable" :key="i.kind" class="mb-3">
        <div class="flex flex-wrap items-center gap-2">
          <NTag size="small" :bordered="false" :type="readable.length < 2 ? 'default' : i.spki === reference ? 'success' : 'error'">
            {{ t(`tools.cert-key-matcher.inputs.${i.kind}`) }}
          </NTag>
          <span class="text-sm">{{ i.description }}</span>
          <span v-if="i.subject" class="text-xs nt-muted font-mono">{{ i.subject }}</span>
        </div>
        <p class="m-0 mt-1 break-all text-xs font-mono">
          SHA-256 {{ i.spki }}
        </p>
      </div>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.cert-key-matcher.how') }}
      </p>
    </CCard>
  </template>
</template>
