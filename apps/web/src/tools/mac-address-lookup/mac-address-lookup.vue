<script setup lang="ts">
import type { OuiTable } from './mac-address-lookup.service'
import { NSpin } from 'naive-ui'
import { computed, onMounted, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { splitList } from '@/utils/ip'
import { loadOuiTable, lookup } from './mac-address-lookup.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'mac', defaultValue: '00:0c:29:3e:1a:7b\n3c:22:fb:12:34:56\n02:42:ac:11:00:02' })
const table = shallowRef<OuiTable>()
onMounted(async () => {
  table.value = await loadOuiTable()
})

const entries = computed(() => splitList(input.value).slice(0, 1000))
const result = computed(() => {
  if (!table.value)
    return { rows: [], invalid: [] as string[] }
  const rows = []
  const invalid: string[] = []
  for (const e of entries.value) {
    const r = lookup(e, table.value)
    if (r)
      rows.push(r)
    else
      invalid.push(e)
  }
  return { rows, invalid }
})
const validation = computed(() => result.value.invalid.length
  ? { isValid: false, status: 'error' as const, message: t('tools.mac-address-lookup.invalid', { count: result.value.invalid.length, entries: result.value.invalid.slice(0, 5).join(', ') }) }
  : undefined)

const rows = computed(() => result.value.rows.map(r => ({
  ...r,
  vendor: r.vendor || (r.administration === 'local' ? t('tools.mac-address-lookup.localNoVendor') : t('tools.mac-address-lookup.unknown')),
  administration: t(`tools.mac-address-lookup.admin.${r.administration}`),
  cast: t(`tools.mac-address-lookup.cast.${r.cast}`),
})))
const k = (key: string) => t(`tools.mac-address-lookup.columns.${key}`)
const columns = computed(() => [
  { key: 'input', title: k('input'), mono: true, width: 170 },
  { key: 'oui', title: k('oui'), mono: true, width: 100 },
  { key: 'vendor', title: k('vendor'), sortable: true, width: 220 },
  { key: 'country', title: k('country'), sortable: true },
  { key: 'administration', title: k('administration') },
  { key: 'cast', title: k('cast') },
])
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.mac-address-lookup.input')" :validation="validation" multiline :rows="5" monospace test-id="input" />
    <p class="m-0 text-sm nt-muted">
      {{ t('tools.mac-address-lookup.hint') }}
    </p>
  </CCard>
  <NSpin v-if="!table" class="w-full" />
  <CCard v-else-if="rows.length" :title="t('common.result')" data-testid="result">
    <ResultTable :columns="columns" :rows="rows" filename="mac-lookup.json" />
  </CCard>
</template>
