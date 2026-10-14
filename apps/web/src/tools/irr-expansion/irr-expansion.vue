<script setup lang="ts">
import type { Platform } from '../router-snippets/router-snippets.service'
import type { IrrResult } from './irr-expansion.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NFormItem, NRadioButton, NRadioGroup, NSelect, NSwitch } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiGet } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { buildQuery } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { asnFilter, PLATFORMS, prefixList } from '../router-snippets/router-snippets.service'
import { isSetName, listName, plain, SOURCES, toEntries } from './irr-expansion.service'

const { t } = useI18n()
const set = useQueryParam({ name: 'set', defaultValue: '' })
const mode = useQueryParam({ name: 'mode', defaultValue: 'prefixes' })
const family = useQueryParam({ name: 'family', defaultValue: '4' })
const aggregate = useQueryParam({ name: 'aggregate', defaultValue: true })
const sources = useQueryParam({ name: 'sources', defaultValue: '' })
const setV = useValidation({ source: set, rules: [{ validator: isSetName, message: t('tools.irr-expansion.invalid') }] })
const format = ref<'plain' | Platform>('plain')
const result = shallowRef<IrrResult>()
const error = ref<ApiErrorType>()
const running = ref(false)
const sourceList = computed({
  get: () => sources.value.split(',').filter(Boolean),
  set: (v: string[]) => (sources.value = v.join(',')),
})
async function run() {
  if (!set.value || !setV.value.isValid)
    return
  running.value = true
  error.value = undefined
  result.value = undefined
  try {
    result.value = await apiGet<IrrResult>(`/v1/bgp/irr${buildQuery({ set: set.value.trim(), mode: mode.value, family: mode.value === 'prefixes' ? family.value : undefined, aggregate: mode.value === 'prefixes' ? aggregate.value : undefined, sources: sources.value || undefined })}`)
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    running.value = false
  }
}
onMounted(() => {
  if (set.value)
    void run()
})
const count = computed(() => result.value ? (result.value.mode === 'asns' ? result.value.asns.length : result.value.prefixes.length) : 0)
const output = computed(() => {
  const r = result.value
  if (!r || count.value === 0)
    return ''
  if (format.value === 'plain')
    return plain(r)
  const name = listName(r.set)
  return r.mode === 'asns' ? asnFilter(r.asns, format.value, name) : prefixList(toEntries(r), format.value, name)
})
const formatOptions = computed(() => [{ label: t('tools.irr-expansion.plain'), value: 'plain' }, ...PLATFORMS.map(p => ({ label: t(`tools.router-snippets.platforms.${p}`), value: p }))])
const sourceOptions = SOURCES.map(s => ({ label: s, value: s }))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="set" :label="t('tools.irr-expansion.set')" placeholder="AS-EXAMPLE" :validation="setV" monospace test-id="set" />
      <div class="flex flex-wrap items-end gap-x-6">
        <NFormItem :label="t('tools.irr-expansion.mode')" :show-feedback="false" class="mb-3">
          <NRadioGroup v-model:value="mode" name="irr-mode">
            <NRadioButton value="prefixes" :label="t('tools.irr-expansion.modes.prefixes')" />
            <NRadioButton value="asns" :label="t('tools.irr-expansion.modes.asns')" />
          </NRadioGroup>
        </NFormItem>
        <template v-if="mode === 'prefixes'">
          <NFormItem :label="t('tools.irr-expansion.family')" :show-feedback="false" class="mb-3">
            <NRadioGroup v-model:value="family" name="irr-family">
              <NRadioButton value="4" label="IPv4" />
              <NRadioButton value="6" label="IPv6" />
            </NRadioGroup>
          </NFormItem>
          <NFormItem :label="t('tools.irr-expansion.aggregate')" :show-feedback="false" class="mb-3">
            <NSwitch v-model:value="aggregate" :aria-label="t('tools.irr-expansion.aggregate')" />
          </NFormItem>
        </template>
      </div>
      <NFormItem :label="t('tools.irr-expansion.sources')" :show-feedback="false" class="mb-3">
        <NSelect v-model:value="sourceList" multiple clearable :options="sourceOptions" :placeholder="t('tools.irr-expansion.allSources')" :aria-label="t('tools.irr-expansion.sources')" />
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!set || !setV.isValid || running" data-testid="run">
        {{ t('tools.irr-expansion.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.irr-expansion.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="result" :title="t(`tools.irr-expansion.found.${result.mode}`, { n: count, set: result.set })" data-testid="result">
    <NAlert v-if="count === 0" type="warning">
      {{ t('tools.irr-expansion.empty', { set: result.set }) }}
    </NAlert>
    <template v-else>
      <NFormItem :label="t('tools.irr-expansion.format')" :show-feedback="false" class="mb-3 max-w-80">
        <NSelect v-model:value="format" :options="formatOptions" :aria-label="t('tools.irr-expansion.format')" />
      </NFormItem>
      <pre class="m-0 mb-4 max-h-120 overflow-auto text-sm font-mono" data-testid="output">{{ output }}</pre>
      <ResultActions :text="output" :json="result" filename="irr-expansion.json" />
    </template>
  </CCard>
</template>
