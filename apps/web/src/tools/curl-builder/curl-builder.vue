<script setup lang="ts">
import type { CurlSpec } from './curl-builder.service'
import { NAlert, NButton, NCheckbox, NFormItem, NInputNumber } from 'naive-ui'
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import CopyButton from '@/components/CopyButton.vue'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { buildCurl, emptySpec, parseCurl } from './curl-builder.service'

const { t } = useI18n()
// Commands often carry tokens and cookies, so nothing is put in the page address.
const pasted = ref('')
const spec = reactive<CurlSpec>(emptySpec())
const headersText = ref('')
const parseError = ref('')

watch(pasted, (v) => {
  if (!v.trim()) {
    parseError.value = ''
    return
  }
  const r = parseCurl(v)
  if (typeof r === 'string') {
    parseError.value = t(`tools.curl-builder.errors.${r}`)
    return
  }
  parseError.value = ''
  Object.assign(spec, r)
  headersText.value = r.headers.map(([k, val]) => `${k}: ${val}`).join('\n')
})
watch(headersText, (v) => {
  spec.headers = v.split('\n').map(l => l.trim()).filter(l => l.includes(':')).map(l => [l.slice(0, l.indexOf(':')).trim(), l.slice(l.indexOf(':') + 1).trim()] as [string, string])
})
const command = computed(() => buildCurl(spec))
const methods = ['GET', 'HEAD', 'POST', 'PUT', 'PATCH', 'DELETE', 'OPTIONS'].map(m => ({ label: m, value: m }))
const httpRequestLink = computed(() => ({ path: '/http-request', query: { url: spec.url, method: spec.method, follow: spec.follow ? 'true' : undefined } }))
</script>

<template>
  <CCard :title="t('tools.curl-builder.parseTitle')">
    <CInputText v-model:value="pasted" :label="t('tools.curl-builder.paste')" placeholder="curl -X POST https://api.example.com -H 'Content-Type: application/json' -d '{&quot;a&quot;:1}'" multiline :rows="4" monospace test-id="paste" :validation="parseError ? { isValid: false, status: 'error', message: parseError } : undefined" />
    <NAlert v-if="spec.unknown.length" type="warning">
      {{ t('tools.curl-builder.unknown', { list: spec.unknown.join(' ') }) }}
    </NAlert>
  </CCard>
  <CCard :title="t('tools.curl-builder.buildTitle')">
    <div class="grid gap-x-4 sm:grid-cols-[8rem_1fr]">
      <CSelect v-model:value="spec.method" :label="t('tools.curl-builder.method')" :options="methods" />
      <CInputText v-model:value="spec.url" :label="t('http.url')" placeholder="https://api.example.com/items" monospace test-id="url" />
    </div>
    <CInputText v-model:value="headersText" :label="t('tools.curl-builder.headers')" placeholder="Content-Type: application/json" multiline :rows="3" monospace test-id="headers" />
    <CInputText v-model:value="spec.data" :label="t('tools.curl-builder.body')" multiline :rows="3" monospace test-id="body" />
    <CInputText v-model:value="spec.user" :label="t('tools.curl-builder.user')" placeholder="user:password" monospace />
    <div class="flex flex-wrap items-center gap-x-6 gap-y-2">
      <NCheckbox v-model:checked="spec.follow">
        {{ t('tools.curl-builder.follow') }}
      </NCheckbox>
      <NCheckbox v-model:checked="spec.compressed">
        {{ t('tools.curl-builder.compressed') }}
      </NCheckbox>
      <NCheckbox v-model:checked="spec.insecure">
        {{ t('tools.curl-builder.insecure') }}
      </NCheckbox>
      <NCheckbox v-model:checked="spec.verbose">
        {{ t('tools.curl-builder.verbose') }}
      </NCheckbox>
      <NFormItem :label="t('tools.curl-builder.maxTime')" :show-feedback="false" label-placement="left" class="mb-0">
        <NInputNumber v-model:value="spec.maxTime" :min="1" :max="3600" clearable :show-button="false" class="w-24" :input-props="{ 'aria-label': t('tools.curl-builder.maxTime') }" />
      </NFormItem>
    </div>
  </CCard>
  <CCard :title="t('tools.curl-builder.commandTitle')" data-testid="result">
    <pre class="m-0 overflow-auto text-sm font-mono" tabindex="0" data-testid="command">{{ command }}</pre>
    <div class="mt-3 flex flex-wrap gap-2">
      <CopyButton :text="command" :label="t('common.copyText')" />
      <NButton v-if="spec.url" tag="a" size="small" secondary :href="$router.resolve(httpRequestLink).href">
        {{ t('tools.curl-builder.openHttpRequest') }}
      </NButton>
    </div>
  </CCard>
</template>
