<script setup lang="ts">
import type { SvcbRecord, SvcbResult } from './svcb-https-viewer.service'
import { NAlert, NButton, NFormItem, NRadioButton, NRadioGroup, NTag } from 'naive-ui'
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
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { isResolver } from '../nat64-dns64-check/nat64-dns64-check.service'
import { isOwnerName, offers } from './svcb-https-viewer.service'

const { t } = useI18n()
const name = useQueryParam({ name: 'name', defaultValue: '' })
const type = useQueryParam({ name: 'type', defaultValue: 'HTTPS' })
const resolver = useQueryParam({ name: 'resolver', defaultValue: '' })
const validation = useValidation({ source: name, rules: [{ validator: (s: string) => !s || isOwnerName(s), message: t('tools.dns-lookup.invalidName') }] })
const resolverValidation = useValidation({ source: resolver, rules: [{ validator: isResolver, message: t('tools.nat64-dns64-check.invalidResolver') }] })
const { data, error, loading, run } = useApi<SvcbResult>('/v1/dns/svcb')
const submit = () => name.value && validation.value.isValid && resolverValidation.value.isValid && run({ name: name.value.trim(), type: type.value, resolver: resolver.value.trim() || 'system' })
onMounted(() => {
  if (name.value)
    void submit()
})
function useDdr() {
  name.value = '_dns.resolver.arpa'
  type.value = 'SVCB'
}
const k = (key: string) => t(`tools.svcb-https-viewer.fields.${key}`)
function rows(r: SvcbRecord) {
  return [
    { label: k('mode'), value: t(`tools.svcb-https-viewer.modes.${r.mode}`), noCopy: true },
    { label: k('target'), value: r.target === '.' ? t('tools.svcb-https-viewer.sameName') : r.target, mono: true },
    { label: 'TTL', value: `${r.ttl} s`, noCopy: true },
    ...r.params.map(p => ({ label: p.key, value: p.value, mono: true })),
  ]
}
const echRows = (r: SvcbRecord) => r.ech.map((c, i) => ({ id: String(i), configId: c.configId, publicName: c.publicName || '—', kem: c.kem || c.version, suites: c.suites.map(s => `${s.kdf} / ${s.aead}`).join(', ') || '—', maxName: c.maxNameLength }))
const c = (key: string) => t(`tools.svcb-https-viewer.columns.${key}`)
const asText = () => (data.value?.records ?? []).map(r => `${r.owner} ${r.priority} ${r.target} ${r.params.map(p => `${p.key}=${p.value}`).join(' ')}`).join('\n')
const empty = computed(() => data.value && !data.value.records.length)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="name" :label="t('tools.svcb-https-viewer.name')" placeholder="cloudflare.com" :validation="validation" monospace test-id="name" />
        <CInputText v-model:value="resolver" :label="t('tools.nat64-dns64-check.resolver')" :placeholder="t('tools.nat64-dns64-check.resolverPlaceholder')" :validation="resolverValidation" monospace test-id="resolver" />
      </div>
      <NFormItem :label="t('tools.svcb-https-viewer.type')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="type" name="type">
          <NRadioButton value="HTTPS" label="HTTPS" />
          <NRadioButton value="SVCB" label="SVCB" />
        </NRadioGroup>
      </NFormItem>
      <div class="flex flex-wrap gap-2">
        <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!name || !validation.isValid" data-testid="run">
          {{ t('tools.svcb-https-viewer.run') }}
        </NButton>
        <NButton quaternary @click="useDdr">
          {{ t('tools.svcb-https-viewer.ddr') }}
        </NButton>
      </div>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <NAlert v-if="empty" type="info" class="mb-4" data-testid="result">
    {{ t('tools.svcb-https-viewer.none', { type: data!.type, name: data!.name, rcode: data!.rcode }) }}
  </NAlert>
  <CCard v-for="(r, i) in data?.records ?? []" :key="i" :title="`${r.owner} · ${r.priority}`" data-testid="result">
    <div class="mb-2 flex flex-wrap gap-1">
      <NTag v-for="o in offers(r)" :key="o" size="small" :bordered="false" type="success">
        {{ o }}
      </NTag>
    </div>
    <CKeyValueList :rows="rows(r)" />
    <template v-if="r.ech.length">
      <h3 class="mb-2 mt-4 text-sm font-semibold">
        {{ t('tools.svcb-https-viewer.ech') }}
      </h3>
      <ResultTable :columns="[{ key: 'configId', title: c('configId'), width: 80 }, { key: 'publicName', title: c('publicName'), mono: true, width: 200 }, { key: 'kem', title: 'KEM', width: 220 }, { key: 'suites', title: c('suites'), width: 300 }, { key: 'maxName', title: c('maxName'), width: 110 }]" :rows="echRows(r)" row-key="id" :actions="false" :label="t('tools.svcb-https-viewer.ech')" />
    </template>
    <NAlert v-if="r.echError" type="warning" class="mt-3">
      {{ r.echError }}
    </NAlert>
  </CCard>
  <div v-if="data?.records.length" class="mb-4">
    <ResultActions :text="asText" :json="data" filename="svcb.json" />
  </div>
</template>
