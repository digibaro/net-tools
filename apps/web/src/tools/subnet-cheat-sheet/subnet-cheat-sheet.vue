<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultTable from '@/components/ResultTable.vue'
import CCard from '@/ui/c-card.vue'
import { ipv4CheatSheet, ipv6CheatSheet } from './subnet-cheat-sheet.service'

const { t, te } = useI18n()
const k = (key: string) => t(`tools.subnet-cheat-sheet.columns.${key}`)
const v4 = ipv4CheatSheet()
const v6 = computed(() => ipv6CheatSheet().map((r) => {
  const key = `tools.subnet-cheat-sheet.v6use.${r.prefix.slice(1)}`
  return { ...r, use: te(key) ? t(key) : '' }
}))
const v4Columns = computed(() => [
  { key: 'prefix', title: k('prefix'), width: 70, mono: true },
  { key: 'netmask', title: k('netmask'), width: 140, mono: true },
  { key: 'wildcard', title: k('wildcard'), width: 140, mono: true },
  { key: 'addresses', title: k('addresses'), width: 140 },
  { key: 'usable', title: k('usable'), width: 140 },
  { key: 'size', title: k('size'), width: 150 },
])
const v6Columns = computed(() => [
  { key: 'prefix', title: k('prefix'), width: 70, mono: true },
  { key: 'subnets64', title: k('subnets64'), width: 200 },
  { key: 'addresses', title: k('addresses'), width: 320 },
  { key: 'use', title: k('use'), width: 260 },
])
</script>

<template>
  <CCard :title="t('tools.subnet-cheat-sheet.v4')" data-testid="v4">
    <ResultTable :columns="v4Columns" :rows="v4" row-key="prefix" :label="t('tools.subnet-cheat-sheet.v4')" filename="ipv4-cheat-sheet.json" />
  </CCard>
  <CCard :title="t('tools.subnet-cheat-sheet.v6')" data-testid="v6">
    <ResultTable :columns="v6Columns" :rows="v6" row-key="prefix" :label="t('tools.subnet-cheat-sheet.v6')" filename="ipv6-cheat-sheet.json" />
  </CCard>
</template>
