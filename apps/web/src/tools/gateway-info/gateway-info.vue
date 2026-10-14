<script setup lang="ts">
import type { GatewayResult } from './gateway-info.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiGet } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { gatewayRows } from './gateway-info.service'

const { t } = useI18n()
const result = shallowRef<GatewayResult>()
const error = ref<ApiErrorType>()
const running = ref(false)
async function run() {
  running.value = true
  error.value = undefined
  result.value = undefined
  try {
    result.value = await apiGet<GatewayResult>('/v1/lan/gateway')
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    running.value = false
  }
}
const rows = computed(() => result.value ? gatewayRows(result.value, (k, p) => t(`tools.gateway-info.${k}`, p ?? {})) : [])
</script>

<template>
  <CCard>
    <NButton type="primary" :loading="running" :disabled="running" data-testid="run" @click="run">
      {{ t('tools.gateway-info.run') }}
    </NButton>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.gateway-info.note') }}
    </p>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <NAlert v-if="result.doubleNat" type="warning" class="mb-3">
      {{ t('tools.gateway-info.doubleNat', { ip: result.externalIp }) }}
    </NAlert>
    <CKeyValueList :rows="rows" />
    <div class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).join('\n')" :json="result" filename="gateway.json" />
    </div>
  </CCard>
</template>
