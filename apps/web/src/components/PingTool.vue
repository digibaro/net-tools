<script setup lang="ts">
import type { components } from '@/api/schema'
import { NButton, NFormItem, NInputNumber } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import PingRun from '@/components/PingRun.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useSse } from '@/composables/useSse'
import { useValidation } from '@/composables/useValidation'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'

type Reply = components['schemas']['PingReply']
type Summary = components['schemas']['PingSummary']

/** Shared page for ICMP ping and TCP ping. */
const props = defineProps<{ kind: 'icmp' | 'tcp', path: string }>()
const { t } = useI18n()
const caps = useCapabilitiesStore()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 443 })
const count = useQueryParam({ name: 'count', defaultValue: 5 })
const interval = useQueryParam({ name: 'interval', defaultValue: 1000 })
const maxCount = computed(() => caps.capabilities?.limits.pingMaxCount ?? 20)
const minInterval = computed(() => caps.capabilities?.limits.pingMinIntervalMs ?? 200)
const validation = useValidation({ source: target, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })

const replies = ref<Reply[]>([])
const summary = shallowRef<Summary>()
const { status, error, start, stop } = useSse(props.path, {
  reply: d => replies.value.push(d as Reply),
  timeout: d => replies.value.push(d as Reply),
  done: d => (summary.value = (d as { summary: Summary }).summary),
})
const running = computed(() => status.value === 'running')
function run() {
  if (!target.value || !validation.value.isValid)
    return
  replies.value = []
  summary.value = undefined
  start({ target: target.value, count: Math.min(count.value, maxCount.value), interval: Math.max(interval.value, minInterval.value), port: props.kind === 'tcp' ? port.value : undefined })
}
onMounted(() => {
  if (target.value)
    run()
})
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4" :class="kind === 'tcp' ? 'sm:grid-cols-[1fr_8rem]' : ''">
        <CInputText v-model:value="target" :label="t('ping.target')" placeholder="example.com" :validation="validation" monospace test-id="target" />
        <NFormItem v-if="kind === 'tcp'" :label="t('tools.tcp-ping.port')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.tcp-ping.port') }" data-testid="port" />
        </NFormItem>
      </div>
      <div class="grid grid-cols-2 max-w-md gap-x-4">
        <NFormItem :label="t('ping.count')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="count" :min="1" :max="maxCount" :show-button="false" :input-props="{ 'aria-label': t('ping.count') }" />
        </NFormItem>
        <NFormItem :label="t('ping.interval')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="interval" :min="minInterval" :max="5000" :step="100" :show-button="false" :input-props="{ 'aria-label': t('ping.interval') }" />
        </NFormItem>
      </div>
      <div class="flex gap-2">
        <NButton type="primary" attr-type="submit" :loading="running" :disabled="!target || !validation.isValid || running" data-testid="run">
          {{ t('ping.run') }}
        </NButton>
        <NButton v-if="running" data-testid="stop" @click="stop">
          {{ t('common.stop') }}
        </NButton>
      </div>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="replies.length || summary" :title="t('common.result')" data-testid="result">
    <PingRun :replies="replies" :summary="summary" :kind="kind" />
  </CCard>
</template>
