<script setup lang="ts">
import type { DnsReverse } from './reverse-dns.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { isIpAddress, verdict } from './reverse-dns.service'

const { t } = useI18n()
const ip = useQueryParam({ name: 'ip', defaultValue: '' })
const resolver = useQueryParam({ name: 'resolver', defaultValue: 'system' })
const validation = useValidation({ source: ip, rules: [{ validator: isIpAddress, message: t('tools.reverse-dns.invalid') }] })
const resolverOptions = computed(() => ['system', 'cloudflare', 'google', 'quad9'].map(v => ({ label: t(`tools.dns-lookup.resolvers.${v}`), value: v })))
const { data, error, loading, run } = useApi<DnsReverse>('/v1/dns/reverse')
const submit = () => ip.value && validation.value.isValid && run({ ip: ip.value, resolver: resolver.value })
onMounted(() => {
  if (ip.value)
    void submit()
})

const status = computed(() => (data.value ? verdict(data.value) : undefined))
const alertType = computed(() => ({ confirmed: 'success', partial: 'warning', unconfirmed: 'error', none: 'info' } as const)[status.value ?? 'none'])
const rows = computed(() => (data.value?.names ?? []).map(n => ({
  name: n.name,
  forward: n.forwardAddresses.join(', ') || '—',
  confirmed: t(n.forwardConfirmed ? 'common.yes' : 'common.no'),
})))
const columns = computed(() => [
  { key: 'name', title: t('tools.reverse-dns.columns.name'), mono: true, width: 260 },
  { key: 'forward', title: t('tools.reverse-dns.columns.forward'), mono: true, width: 300 },
  { key: 'confirmed', title: t('tools.reverse-dns.columns.confirmed'), width: 130 },
])
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_14rem]">
        <CInputText v-model:value="ip" :label="t('tools.reverse-dns.ip')" placeholder="8.8.8.8" :validation="validation" monospace test-id="ip" />
        <CSelect v-model:value="resolver" :label="t('tools.dns-lookup.resolver')" :options="resolverOptions" />
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ip || !validation.isValid" data-testid="run">
        {{ t('tools.reverse-dns.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <NAlert :type="alertType" class="mb-3" data-testid="verdict">
      {{ t(`tools.reverse-dns.verdict.${status}`) }}
    </NAlert>
    <CKeyValueList :rows="[{ label: t('tools.reverse-dns.ptrName'), value: data.ptrName, mono: true }, { label: t('dns.resolver'), value: `${data.resolver} · ${data.rcode} · ${data.rttMs} ms`, noCopy: true }]" />
    <ResultTable v-if="rows.length" class="mt-4" :columns="columns" :rows="rows" filename="reverse-dns.json" />
  </CCard>
</template>
