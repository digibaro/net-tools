<script setup lang="ts">
import type { NtpCheck } from './ntp-check.service'
import { NAlert, NButton, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isValidHost, NTP_PRESETS, offsetQuality } from './ntp-check.service'

const { t } = useI18n()
const server = useQueryParam({ name: 'server', defaultValue: '' })
const validation = useValidation({ source: server, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const { data, error, loading, run } = useApi<NtpCheck>('/v1/ntp')
const submit = () => server.value && validation.value.isValid && run({ server: server.value })
function pick(s: string) {
  server.value = s
  void submit()
}
onMounted(() => {
  if (server.value)
    void submit()
})
const rows = computed(() => {
  const r = data.value
  if (!r)
    return []
  const k = (key: string) => t(`tools.ntp-check.fields.${key}`)
  return [
    { label: k('address'), value: r.address, mono: true },
    { label: k('offset'), value: `${r.offsetMs} ms`, noCopy: true },
    { label: k('rtt'), value: `${r.rttMs} ms`, noCopy: true },
    { label: k('stratum'), value: String(r.stratum), noCopy: true },
    { label: k('refid'), value: r.referenceId, mono: true },
    { label: k('referenceTime'), value: r.referenceTime, mono: true },
    { label: k('serverTime'), value: r.serverTime, mono: true },
    { label: k('rootDelay'), value: `${r.rootDelayMs} ms`, noCopy: true },
    { label: k('rootDispersion'), value: `${r.rootDispersionMs} ms`, noCopy: true },
    { label: k('precision'), value: `${r.precisionMs} ms`, noCopy: true },
    { label: k('leap'), value: r.leap, noCopy: true },
    { label: k('version'), value: `NTPv${r.version}`, noCopy: true },
  ]
})
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="server" :label="t('tools.ntp-check.server')" placeholder="pool.ntp.org" :validation="validation" monospace test-id="server" />
      <div class="mb-3 flex flex-wrap gap-1">
        <NTag v-for="p in NTP_PRESETS" :key="p" size="small" class="cursor-pointer" :bordered="false" @click="pick(p)">
          {{ p }}
        </NTag>
      </div>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!server || !validation.isValid" data-testid="run">
        {{ t('tools.ntp-check.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert :type="!data.valid ? 'error' : offsetQuality(data.offsetMs) === 'good' ? 'success' : 'warning'" class="mb-4" data-testid="verdict">
      {{ data.valid ? t(`tools.ntp-check.quality.${offsetQuality(data.offsetMs)}`, { offset: data.offsetMs }) : t('tools.ntp-check.invalid', { problem: data.problem }) }}
    </NAlert>
    <CCard :title="t('common.result')" data-testid="result">
      <CKeyValueList :rows="rows" />
      <div class="mt-4">
        <ResultActions :text="asText" :json="data" filename="ntp.json" />
      </div>
    </CCard>
  </template>
</template>
