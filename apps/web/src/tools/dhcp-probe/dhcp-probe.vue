<script setup lang="ts">
import type { DhcpResult } from './dhcp-probe.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NFormItem, NInputNumber } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiGet } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { buildQuery } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { formatLease, isInterfaceName, verdict } from './dhcp-probe.service'

const { t } = useI18n()
const iface = useQueryParam({ name: 'interface', defaultValue: '' })
const seconds = useQueryParam({ name: 'seconds', defaultValue: 4 })
const ifaceV = useValidation({ source: iface, rules: [{ validator: isInterfaceName, message: t('tools.dhcp-probe.invalidInterface') }] })
const result = shallowRef<DhcpResult>()
const error = ref<ApiErrorType>()
const running = ref(false)
async function run() {
  if (!ifaceV.value.isValid)
    return
  running.value = true
  error.value = undefined
  result.value = undefined
  try {
    result.value = await apiGet<DhcpResult>(`/v1/lan/dhcp${buildQuery({ interface: iface.value.trim() || undefined, seconds: seconds.value })}`)
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    running.value = false
  }
}
const alertType = { none: 'warning', single: 'success', multiple: 'error' } as const
const rows = computed(() => (result.value?.offers ?? []).map(o => ({
  id: `${o.server}|${o.from}|${o.offeredIp}`,
  server: o.server || o.from,
  relay: o.relay || '—',
  offered: o.subnetMask ? `${o.offeredIp} / ${o.subnetMask}` : o.offeredIp,
  routers: o.routers.join(', ') || '—',
  dns: o.dns.join(', ') || '—',
  domain: o.domain || '—',
  lease: formatLease(o.leaseSeconds),
})))
const c = (k: string) => t(`tools.dhcp-probe.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="iface" :label="t('tools.dhcp-probe.interface')" placeholder="eth0" :validation="ifaceV" monospace test-id="interface" />
        <NFormItem :label="t('tools.dhcp-probe.seconds')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="seconds" :min="1" :max="10" :show-button="false" :input-props="{ 'aria-label': t('tools.dhcp-probe.seconds') }" />
        </NFormItem>
      </div>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="running || !ifaceV.isValid" data-testid="run">
        {{ t('tools.dhcp-probe.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.dhcp-probe.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <NAlert :type="alertType[verdict(result)]" class="mb-3">
      {{ t(`tools.dhcp-probe.verdict.${verdict(result)}`, { n: result.servers, iface: result.interface, seconds: result.seconds }) }}
    </NAlert>
    <ResultTable v-if="rows.length" :columns="[{ key: 'server', title: c('server'), mono: true, width: 140 }, { key: 'relay', title: c('relay'), mono: true, width: 120 }, { key: 'offered', title: c('offered'), mono: true, width: 230 }, { key: 'routers', title: c('routers'), mono: true, width: 140 }, { key: 'dns', title: c('dns'), mono: true, width: 180 }, { key: 'domain', title: c('domain'), width: 120 }, { key: 'lease', title: c('lease'), width: 80 }]" :rows="rows" row-key="id" :actions="false" />
    <div class="mt-3">
      <ResultActions :text="() => rows.map(r => `${r.server}\t${r.offered}\t${r.routers}\t${r.dns}`).join('\n')" :json="result" filename="dhcp-probe.json" />
    </div>
  </CCard>
</template>
