<script setup lang="ts">
import type { Exposure } from './passive-exposure.service'
import { NAlert, NButton, NTag } from 'naive-ui'
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
import { cveUrl, isIpAddress } from './passive-exposure.service'

const { t } = useI18n()
const ip = useQueryParam({ name: 'ip', defaultValue: '' })
const validation = useValidation({ source: ip, rules: [{ validator: isIpAddress, message: t('tools.reverse-dns.invalid') }] })
const { data, error, loading, run } = useApi<Exposure>('/v1/reputation/internetdb')
const submit = () => ip.value && validation.value.isValid && run({ ip: ip.value.trim() })
onMounted(() => {
  if (ip.value)
    void submit()
})
const rows = computed(() => (data.value?.found
  ? [
      { label: t('tools.passive-exposure.ports'), value: data.value.ports.join(', ') || '—', mono: true },
      { label: t('tools.passive-exposure.hostnames'), value: data.value.hostnames.join(', ') || '—', mono: true },
      { label: t('tools.passive-exposure.software'), value: data.value.cpes.join(', ') || '—', mono: true },
    ]
  : []))
const vulnRows = computed(() => (data.value?.vulns ?? []).map(v => ({ id: v, link: cveUrl(v) })))
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="submit">
      <CInputText v-model:value="ip" :label="t('tools.reverse-dns.ip')" placeholder="1.1.1.1" :validation="validation" monospace test-id="ip" />
      <NButton type="primary" attr-type="submit" :loading="loading" :disabled="!ip || !validation.isValid" data-testid="run">
        {{ t('bgp.lookUp') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.passive-exposure.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <template v-if="data">
    <NAlert v-if="!data.found" type="info" class="mb-4" data-testid="not-found">
      {{ t('tools.passive-exposure.notFound', { ip: data.ip }) }}
    </NAlert>
    <CCard v-else :title="data.ip" data-testid="result">
      <div v-if="data.tags.length" class="mb-3 flex flex-wrap gap-1">
        <NTag v-for="tag in data.tags" :key="tag" size="small" :bordered="false">
          {{ tag }}
        </NTag>
      </div>
      <CKeyValueList :rows="rows" />
      <template v-if="vulnRows.length">
        <NAlert type="warning" class="my-3">
          {{ t('tools.passive-exposure.vulns', { n: vulnRows.length }) }}
        </NAlert>
        <ResultTable :columns="[{ key: 'id', title: 'CVE', mono: true, hrefKey: 'link' }]" :rows="vulnRows" row-key="id" :max-height="320" :actions="false" />
      </template>
      <div class="mt-3">
        <ResultActions :text="() => rows.map(r => `${r.label}: ${r.value}`).concat(data!.vulns).join('\n')" :json="data" filename="exposure.json" />
      </div>
    </CCard>
  </template>
</template>
