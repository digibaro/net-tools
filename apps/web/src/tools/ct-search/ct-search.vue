<script setup lang="ts">
import type { CtSearch } from './ct-search.service'
import { NButton, NFormItem, NSwitch, NTag } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ApiError from '@/components/ApiError.vue'
import CopyButton from '@/components/CopyButton.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useApi } from '@/composables/useApi'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidQueryName } from '../dns-lookup/dns-lookup.service'
import { splitNames } from './ct-search.service'

const { t } = useI18n()
const domain = useQueryParam({ name: 'domain', defaultValue: '' })
const expired = useQueryParam({ name: 'expired', defaultValue: false })
const validation = useValidation({ source: domain, rules: [{ validator: isValidQueryName, message: t('tools.dns-lookup.invalidName') }] })
const { data, error, loading, run } = useApi<CtSearch>('/v1/ct')
const submit = () => domain.value && validation.value.isValid && run({ domain: domain.value.trim(), includeExpired: expired.value || undefined })
onMounted(() => {
  if (domain.value)
    void submit()
})
const names = computed(() => splitNames(data.value?.names ?? []))
const certRows = computed(() => (data.value?.certs ?? []).map(c => ({ id: c.id, link: c.url, issuer: c.issuer, names: c.names.join(', '), from: c.notBefore.slice(0, 10), until: c.notAfter.slice(0, 10) + (c.expired ? ` (${t('tools.ct-search.expired')})` : '') })))
const c = (key: string) => t(`tools.ct-search.columns.${key}`)
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="domain" :label="t('email.domain')" placeholder="example.com" :validation="validation" monospace test-id="domain" />
      <NFormItem :label="t('tools.ct-search.includeExpired')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="expired" :aria-label="t('tools.ct-search.includeExpired')" />
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!domain || !validation.isValid" data-testid="run">
        {{ t('tools.ct-search.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.ct-search.slowNote') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <CCard :title="t('tools.ct-search.namesTitle', { n: data.names.length })" data-testid="result">
      <p class="mt-0 text-sm">
        {{ t('tools.ct-search.summary', { certs: data.total, names: data.names.length }) }}
      </p>
      <div class="mb-3 flex flex-wrap gap-1">
        <NTag v-for="i in data.issuers" :key="i.name" size="small" :bordered="false">
          {{ i.name }} · {{ i.count }}
        </NTag>
      </div>
      <pre class="m-0 max-h-80 overflow-auto text-sm font-mono" tabindex="0" data-testid="names">{{ [...names.hosts, ...names.wildcards].join('\n') }}</pre>
      <div class="mt-3">
        <CopyButton :text="() => names.hosts.join('\n')" :label="t('tools.ct-search.copyHosts')" />
      </div>
    </CCard>
    <CCard v-if="certRows.length" :title="t('tools.ct-search.certsTitle')">
      <ResultTable :columns="[{ key: 'id', title: 'crt.sh', hrefKey: 'link', width: 120 }, { key: 'issuer', title: c('issuer'), width: 200 }, { key: 'names', title: c('names'), mono: true, width: 320 }, { key: 'from', title: c('from'), mono: true, width: 110 }, { key: 'until', title: c('until'), mono: true, width: 160 }]" :rows="certRows" row-key="id" :max-height="480" filename="ct-certificates.json" />
      <p v-if="data.truncated" class="mb-0 text-xs nt-muted">
        {{ t('tools.ct-search.truncated') }}
      </p>
    </CCard>
  </template>
</template>
