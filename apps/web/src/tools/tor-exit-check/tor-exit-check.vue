<script setup lang="ts">
import type { TorCheck } from './tor-exit-check.service'
import { NAlert, NButton } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isIpAddress, verdict } from './tor-exit-check.service'

const { t } = useI18n()
const ip = useQueryParam({ name: 'ip', defaultValue: '' })
const validation = useValidation({ source: ip, rules: [{ validator: isIpAddress, message: t('tools.reverse-dns.invalid') }] })
const { data, error, loading, run } = useApi<TorCheck>('/v1/tor-exit')
const submit = () => ip.value && validation.value.isValid && run({ ip: ip.value.trim() })
onMounted(() => {
  if (ip.value)
    void submit()
})
const alertType = { exit: 'warning', relay: 'info', none: 'success' } as const
const relayRows = computed(() => (data.value?.relays ?? []).map(r => ({ nickname: r.nickname, link: r.url, flags: r.flags.join(' '), network: `${r.as} ${r.asName}`.trim(), country: r.country, seen: `${r.firstSeen.slice(0, 10)} – ${r.lastSeen.slice(0, 10)}`, running: r.running ? t('common.yes') : t('common.no') })))
const c = (key: string) => t(`tools.tor-exit-check.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="ip" :label="t('tools.reverse-dns.ip')" placeholder="185.220.101.1" :validation="validation" monospace test-id="ip" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ip || !validation.isValid" data-testid="run">
        {{ t('tools.tor-exit-check.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="alertType[verdict(data)]" class="mb-4" data-testid="result">
      {{ t(`tools.tor-exit-check.verdict.${verdict(data)}`, { ip: data.ip }) }}
    </NAlert>
    <CCard v-if="relayRows.length" :title="t('tools.tor-exit-check.relays')">
      <ResultTable :columns="[{ key: 'nickname', title: c('nickname'), hrefKey: 'link', width: 160 }, { key: 'flags', title: c('flags'), width: 220 }, { key: 'network', title: c('network'), width: 240 }, { key: 'country', title: c('country'), width: 80 }, { key: 'seen', title: c('seen'), mono: true, width: 220 }, { key: 'running', title: c('running'), width: 90 }]" :rows="relayRows" row-key="nickname" filename="tor-relays.json" />
    </CCard>
    <p v-if="data.relayError" class="text-xs nt-muted">
      {{ t('tools.tor-exit-check.relayError', { error: data.relayError }) }}
    </p>
    <ResultActions :text="() => `${data!.ip}: ${verdict(data!)}`" :json="data" filename="tor-check.json" />
    <p class="text-xs nt-muted">
      {{ t('tools.tor-exit-check.source', { n: data.listSize }) }}
    </p>
  </template>
</template>
