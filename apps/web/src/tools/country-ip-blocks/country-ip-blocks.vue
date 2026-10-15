<script setup lang="ts">
import type { CountryBlocks, Format } from './country-ip-blocks.service'
import { NButton, NFormItem, NInput, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import { countryName } from '@/components/remote'
import ResultActions from '@/components/ResultActions.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import { flag } from '@/tools/ip-geolocation/ip-geolocation.service'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import CSelect from '@/ui/c-select.vue'
import { formatCount } from '@/utils/ip'
import { FORMATS, isCountryCode, render } from './country-ip-blocks.service'

const { t, locale } = useI18n()
const country = useQueryParam({ name: 'country', defaultValue: '' })
const family = useQueryParam({ name: 'family', defaultValue: 'all' })
const format = useQueryParam({ name: 'format', defaultValue: 'plain' })
const validation = useValidation({ source: country, rules: [{ validator: (s: string) => !s || isCountryCode(s), message: t('tools.country-ip-blocks.invalid') }] })
const cc = computed(() => country.value.trim().toUpperCase())
const preview = computed(() => (isCountryCode(cc.value) ? `${flag(cc.value)} ${countryName(cc.value, locale.value)}` : ''))
const { data, error, loading, run } = useApi<CountryBlocks>('/v1/ip/country-blocks')
const submit = () => isCountryCode(cc.value) && run({ country: cc.value, family: family.value })
onMounted(() => {
  if (isCountryCode(cc.value))
    void submit()
})
const prefixes = computed(() => (data.value ? [...data.value.ipv4, ...data.value.ipv6] : []))
const output = computed(() => (data.value ? render(prefixes.value, format.value as Format, `COUNTRY-${data.value.country}`) : ''))
const formatOptions = computed(() => FORMATS.map(f => ({ label: t(`tools.country-ip-blocks.formats.${f}`), value: f })))
const k = (key: string) => t(`tools.country-ip-blocks.fields.${key}`)
const rows = computed(() => data.value
  ? [
      { label: k('country'), value: `${flag(data.value.country)} ${countryName(data.value.country, locale.value)} (${data.value.country})`, noCopy: true },
      ...(family.value !== '6' ? [{ label: k('ipv4'), value: t('tools.country-ip-blocks.v4Summary', { prefixes: formatCount(BigInt(data.value.ipv4.length)), entries: formatCount(BigInt(data.value.ipv4Entries)), addresses: formatCount(BigInt(data.value.ipv4Addresses)) }), noCopy: true }] : []),
      ...(family.value !== '4' ? [{ label: k('ipv6'), value: t('tools.country-ip-blocks.v6Summary', { prefixes: formatCount(BigInt(data.value.ipv6.length)), entries: formatCount(BigInt(data.value.ipv6Entries)), n48: formatCount(BigInt(data.value.ipv6Slash48s)) }), noCopy: true }] : []),
      { label: k('asns'), value: formatCount(BigInt(data.value.asns)), noCopy: true },
      { label: k('source'), value: t('tools.country-ip-blocks.source', { time: data.value.queryTime }), noCopy: true },
    ]
  : [])
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid items-start gap-x-4 sm:grid-cols-[10rem_1fr]">
        <CInputText v-model:value="country" :label="t('tools.country-ip-blocks.country')" placeholder="NL" :validation="validation" monospace test-id="country" />
        <p class="m-0 mt-8 text-sm" data-testid="country-name">
          {{ preview }}
        </p>
      </div>
      <NFormItem :label="t('tools.country-ip-blocks.family')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="family" name="family">
          <NRadioButton value="all" :label="t('tools.country-ip-blocks.both')" />
          <NRadioButton value="4" label="IPv4" />
          <NRadioButton value="6" label="IPv6" />
        </NRadioGroup>
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!isCountryCode(cc)" data-testid="run">
        {{ t('tools.country-ip-blocks.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="data" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <CSelect v-model:value="format" :label="t('tools.country-ip-blocks.format')" :options="formatOptions" class="mt-3" test-id="format" />
    <NInput :value="output" type="textarea" readonly :rows="12" class="font-mono" :input-props="{ 'aria-label': t('tools.country-ip-blocks.output') }" data-testid="output" />
    <div class="mt-3">
      <ResultActions :text="output" :json="data" :filename="`country-${data.country}.json`" />
    </div>
  </CCard>
</template>
