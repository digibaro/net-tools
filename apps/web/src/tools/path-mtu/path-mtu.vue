<script setup lang="ts">
import type { PmtuProbe, PmtuResult } from './path-mtu.service'
import { NAlert, NButton, NFormItem, NRadioButton, NRadioGroup, NTag } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidHost } from '@/utils/host'
import { likelyCause } from './path-mtu.service'

const { t } = useI18n()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const max = useQueryParam({ name: 'max', defaultValue: 1500 })
const validation = useValidation({ source: target, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const probes = ref<PmtuProbe[]>([])
const result = shallowRef<PmtuResult>()
const { status, error, start } = useSse('/v1/pmtu/stream', {
  probe: d => probes.value.push(d as PmtuProbe),
  done: d => (result.value = (d as { summary: PmtuResult }).summary),
})
const running = computed(() => status.value === 'running')
function run() {
  if (!target.value || !validation.value.isValid)
    return
  probes.value = []
  result.value = undefined
  start({ target: target.value.trim(), max: max.value })
}
onMounted(() => {
  if (target.value)
    run()
})
const rows = computed(() => (result.value?.reachable
  ? [
      { label: t('tools.path-mtu.fields.mtu'), value: `${result.value.pathMtu} bytes` },
      { label: t('tools.path-mtu.fields.mss'), value: `${result.value.mss} bytes` },
      { label: t('tools.path-mtu.fields.address'), value: result.value.address, mono: true },
      { label: t('tools.path-mtu.fields.probes'), value: String(result.value.probes), noCopy: true },
    ]
  : []))
const cause = computed(() => (result.value ? likelyCause(result.value.pathMtu) : undefined))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="target" :label="t('ping.target')" placeholder="example.com" :validation="validation" monospace test-id="target" />
      <NFormItem :label="t('tools.path-mtu.max')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="max" name="max">
          <NRadioButton :value="1500" label="1500" />
          <NRadioButton :value="9000" :label="t('tools.path-mtu.jumbo')" />
        </NRadioGroup>
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!target || !validation.isValid || running" data-testid="run">
        {{ t('tools.path-mtu.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="probes.length" :title="t('common.result')" data-testid="result">
    <NAlert v-if="result && !result.reachable" type="warning" class="mb-3">
      {{ result.note }}
    </NAlert>
    <template v-else-if="result">
      <NAlert type="info" class="mb-3" data-testid="mtu">
        {{ t('tools.path-mtu.found', { mtu: result.pathMtu }) }}
        <template v-if="cause">
          {{ t(`tools.path-mtu.causes.${cause}`) }}
        </template>
      </NAlert>
      <CKeyValueList :rows="rows" />
    </template>
    <h3 class="mb-1 mt-4 text-sm font-semibold">
      {{ t('tools.path-mtu.probes') }}
    </h3>
    <div class="flex flex-wrap gap-1">
      <NTag v-for="(p, i) in probes" :key="i" size="small" :bordered="false" :type="p.ok ? 'success' : 'error'">
        {{ p.mtu }} {{ p.ok ? '✓' : '✗' }}
      </NTag>
    </div>
    <div v-if="result" class="mt-3">
      <ResultActions :text="() => `${result!.target} (${result!.address}): path MTU ${result!.pathMtu}, MSS ${result!.mss}`" :json="{ result, probes }" filename="path-mtu.json" />
    </div>
  </CCard>
</template>
