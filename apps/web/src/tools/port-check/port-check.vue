<script setup lang="ts">
import type { PortCheck } from './port-check.service'
import { NButton, NFormItem, NRadioButton, NRadioGroup, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost, isValidPorts, STATE_TYPE } from './port-check.service'

const { t } = useI18n()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const ports = useQueryParam({ name: 'ports', defaultValue: '22,80,443' })
const proto = useQueryParam({ name: 'proto', defaultValue: 'tcp' })
const targetValidation = useValidation({ source: target, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const portsValidation = useValidation({ source: ports, rules: [{ validator: isValidPorts, message: t('tools.port-check.invalidPorts') }] })
const { data, error, loading, run } = useApi<PortCheck>('/v1/port/check')
const canRun = computed(() => !!target.value && targetValidation.value.isValid && portsValidation.value.isValid)
const submit = () => canRun.value && run({ target: target.value, ports: ports.value, proto: proto.value })
onMounted(() => {
  if (target.value)
    void submit()
})
const asText = () => (data.value?.rows ?? []).map(r => `${r.port}/${r.proto}\t${r.state}\t${r.service}\t${r.rttMs ? `${r.rttMs} ms` : r.detail}`).join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_12rem]">
        <CInputText v-model:value="target" :label="t('ping.target')" placeholder="example.com" :validation="targetValidation" monospace test-id="target" />
        <CInputText v-model:value="ports" :label="t('tools.port-check.ports')" placeholder="22,80,443" :validation="portsValidation" monospace test-id="ports" />
      </div>
      <NFormItem :label="t('trace.protocol')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="proto" name="proto">
          <NRadioButton value="tcp" label="TCP" />
          <NRadioButton value="udp" :label="t('tools.port-check.udp')" />
        </NRadioGroup>
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!canRun" data-testid="run">
        {{ t('tools.port-check.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('tools.port-check.result', { address: data.address })" data-testid="result">
    <ul class="m-0 list-none p-0">
      <li v-for="r in data.rows" :key="`${r.proto}${r.port}`" class="flex flex-wrap items-center gap-2 py-1.5" :data-testid="`port-${r.port}`">
        <span class="w-24 font-mono">{{ r.port }}/{{ r.proto }}</span>
        <NTag :type="STATE_TYPE[r.state]" size="small" :bordered="false">
          {{ t(`tools.port-check.states.${r.state}`) }}
        </NTag>
        <span v-if="r.service" class="text-sm nt-muted">{{ r.service }}</span>
        <span class="text-sm nt-muted">{{ r.rttMs ? `${r.rttMs} ms` : r.detail }}</span>
      </li>
    </ul>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.port-check.vantage') }}
    </p>
    <div class="mt-3">
      <ResultActions :text="asText" :json="data" filename="port-check.json" />
    </div>
  </CCard>
</template>
