<script setup lang="ts">
import type { Dhcpv6Result } from './dhcpv6-probe.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NFormItem, NInputNumber } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiGet } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { buildQuery } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { lifetime } from '../ipv6-neighbors/ipv6-neighbors.service'
import { isInterfaceName } from '../lldp-neighbors/lldp-neighbors.service'
import { verdict } from './dhcpv6-probe.service'

const { t } = useI18n()
const iface = useQueryParam({ name: 'interface', defaultValue: '' })
const seconds = useQueryParam({ name: 'seconds', defaultValue: 4 })
const ifaceV = useValidation({ source: iface, rules: [{ validator: isInterfaceName, message: t('tools.dhcpv6-probe.invalidInterface') }] })
const result = shallowRef<Dhcpv6Result>()
const error = ref<ApiErrorType>()
const running = ref(false)
async function run() {
  if (!ifaceV.value.isValid)
    return
  running.value = true
  error.value = undefined
  result.value = undefined
  try {
    result.value = await apiGet<Dhcpv6Result>(`/v1/lan/dhcpv6${buildQuery({ interface: iface.value.trim() || undefined, seconds: seconds.value })}`)
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
  id: `${o.duid}|${o.server}`,
  server: o.server,
  duid: o.duid || '—',
  addresses: o.addresses.map(a => `${a.value} (${lifetime(a.validSeconds)})`).join(', ') || '—',
  prefixes: o.prefixes.map(p => `${p.value} (${lifetime(p.validSeconds)})`).join(', ') || '—',
  dns: o.dns.join(', ') || '—',
  search: o.search.join(', ') || '—',
  preference: o.preference,
  status: o.status || 'OK',
})))
const c = (k: string) => t(`tools.dhcpv6-probe.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
        <CInputText v-model:value="iface" :label="t('tools.dhcpv6-probe.interface')" placeholder="eth0" :validation="ifaceV" monospace test-id="interface" />
        <NFormItem :label="t('tools.dhcpv6-probe.seconds')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="seconds" :min="1" :max="10" :show-button="false" :input-props="{ 'aria-label': t('tools.dhcpv6-probe.seconds') }" />
        </NFormItem>
      </div>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="running || !ifaceV.isValid" data-testid="run">
        {{ t('tools.dhcpv6-probe.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.dhcpv6-probe.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <NAlert :type="alertType[verdict(result)]" class="mb-3">
      {{ t(`tools.dhcpv6-probe.verdict.${verdict(result)}`, { n: result.servers, iface: result.interface, seconds: result.seconds }) }}
    </NAlert>
    <ResultTable v-if="rows.length" :columns="[{ key: 'server', title: c('server'), mono: true, width: 200 }, { key: 'addresses', title: c('addresses'), mono: true, width: 240 }, { key: 'prefixes', title: c('prefixes'), mono: true, width: 240 }, { key: 'dns', title: c('dns'), mono: true, width: 200 }, { key: 'search', title: c('search'), width: 140 }, { key: 'preference', title: c('preference'), width: 90 }, { key: 'duid', title: 'DUID', mono: true, width: 240 }, { key: 'status', title: c('status'), width: 140 }]" :rows="rows" row-key="id" filename="dhcpv6-probe.json" />
  </CCard>
</template>
