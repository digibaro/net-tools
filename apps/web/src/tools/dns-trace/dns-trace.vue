<script setup lang="ts">
import type { TraceStep, TraceSummary } from './dns-trace.service'
import { NButton, NTag } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import StreamConsole from '@/components/StreamConsole.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { isValidQueryName, RECORD_TYPES } from '../dns-lookup/dns-lookup.service'
import { traceText } from './dns-trace.service'

const { t } = useI18n()
const name = useQueryParam({ name: 'name', defaultValue: '' })
const type = useQueryParam({ name: 'type', defaultValue: 'A' })
const validation = useValidation({ source: name, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const typeOptions = RECORD_TYPES.map(v => ({ label: v, value: v }))

const steps = ref<TraceStep[]>([])
const summary = shallowRef<TraceSummary>()
const { status, error, start, stop } = useSse('/v1/dns/trace/stream', {
  step: d => steps.value.push(d as TraceStep),
  done: d => (summary.value = (d as { summary: TraceSummary }).summary),
})
function run() {
  if (!name.value || !validation.value.isValid)
    return
  steps.value = []
  summary.value = undefined
  start({ name: name.value, type: type.value })
}
onMounted(() => {
  if (name.value)
    run()
})
const running = computed(() => status.value === 'running')
const lines = computed(() => traceText(steps.value).split('\n'))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_9rem]">
        <CInputText v-model:value="name" :label="t('tools.dns-lookup.name')" placeholder="www.example.com" :validation="validation" monospace test-id="name" />
        <CSelect v-model:value="type" :label="t('tools.dns-lookup.type')" :options="typeOptions" filterable />
      </div>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="!name || !validation.isValid || running" data-testid="run">
          {{ t('tools.dns-trace.run') }}
        </NButton>
        <NButton v-if="running" data-testid="stop" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="steps.length" :title="t('tools.dns-trace.steps', { count: steps.length })" data-testid="result">
    <template #header-extra>
      <NTag v-if="summary" :type="summary.rcode === 'NOERROR' && summary.answers ? 'success' : 'warning'" :bordered="false" data-testid="summary">
        {{ t('tools.dns-trace.summary', { rcode: summary.rcode, answers: summary.answers }) }}
      </NTag>
    </template>
    <StreamConsole :lines="lines" max-height="32rem" />
    <div class="mt-3">
      <ResultActions :text="() => lines.join('\n')" :json="steps" filename="dns-trace.json" />
    </div>
  </CCard>
</template>
