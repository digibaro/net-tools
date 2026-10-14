<script setup lang="ts">
import type { AsnLookup } from './ip-to-asn.service'
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
import { isIpAddress, rirName } from './ip-to-asn.service'

const { t } = useI18n()
const ip = useQueryParam({ name: 'ip', defaultValue: '' })
const validation = useValidation({ source: ip, rules: [{ validator: isIpAddress, message: t('tools.reverse-dns.invalid') }] })
const { data, error, loading, run } = useApi<AsnLookup>('/v1/asn/lookup')
const submit = () => ip.value && validation.value.isValid && run({ ip: ip.value })
onMounted(() => {
  if (ip.value)
    void submit()
})
const rows = computed(() => (data.value?.origins ?? []).map(o => ({ asn: `AS${o.asn}`, name: o.asName, prefix: o.prefix, country: o.country, registry: rirName(o.registry), allocated: o.allocated })))
const k = (key: string) => t(`tools.ip-to-asn.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="ip" :label="t('tools.reverse-dns.ip')" placeholder="8.8.8.8" :validation="validation" monospace test-id="ip" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ip || !validation.isValid" data-testid="run">
        {{ t('tools.ip-to-asn.run') }}
      </NButton>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-if="!data.announced" type="info" class="mb-4" data-testid="not-announced">
      {{ t('tools.ip-to-asn.notAnnounced', { ip: data.ip }) }}
    </NAlert>
    <CCard v-else :title="t('common.result')" data-testid="result">
      <ResultTable :columns="[{ key: 'asn', title: k('asn'), mono: true, width: 100 }, { key: 'name', title: k('name'), width: 220 }, { key: 'prefix', title: k('prefix'), mono: true, width: 180 }, { key: 'country', title: k('country'), width: 80 }, { key: 'registry', title: k('registry'), width: 110 }, { key: 'allocated', title: k('allocated'), width: 120 }]" :rows="rows" filename="asn.json" />
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.ip-to-asn.source') }}
      </p>
    </CCard>
  </template>
</template>
