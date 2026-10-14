<script setup lang="ts">
import type { ApiError as ApiErrorType, EchoInfo } from '@/api/client'
import { Refresh } from '@vicons/tabler'
import { NAlert, NButton, NIcon, NSpin } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getEcho } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultTable from '@/components/ResultTable.vue'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { headerRows, proxyHints } from './request-headers.service'

const { t } = useI18n()
const echo = shallowRef<EchoInfo>()
const error = shallowRef<ApiErrorType>()
const loading = ref(false)

async function load() {
  loading.value = true
  error.value = undefined
  try {
    echo.value = await getEcho()
  }
  catch (e) {
    error.value = e as ApiErrorType
  }
  finally {
    loading.value = false
  }
}
onMounted(load)

const rows = computed(() => echo.value
  ? headerRows(echo.value).map(r => ({ ...r, note: r.addedByNetTools ? t('tools.request-headers.addedByNetTools') : '' }))
  : [])
const hints = computed(() => (echo.value ? proxyHints(echo.value) : []))
const summary = computed(() => echo.value
  ? [
      { label: t('tools.request-headers.fields.ip'), value: echo.value.ip, mono: true },
      { label: t('tools.request-headers.fields.protocol'), value: `${echo.value.protocol} (${echo.value.scheme})`, noCopy: true },
      { label: t('tools.request-headers.fields.method'), value: echo.value.method, noCopy: true },
    ]
  : [])
const k = (key: string) => t(`tools.request-headers.columns.${key}`)
const columns = computed(() => [
  { key: 'name', title: k('name'), width: 200, mono: true, sortable: true },
  { key: 'value', title: k('value'), width: 420, mono: true },
  { key: 'note', title: k('note'), width: 180 },
])
</script>

<template>
  <div class="mb-4 flex justify-end">
    <NButton :loading="loading" @click="load">
      <template #icon>
        <NIcon aria-hidden="true" :component="Refresh" />
      </template>
      {{ t('tools.request-headers.refresh') }}
    </NButton>
  </div>
  <ApiError v-if="error" :error="error" />
  <NSpin :show="loading && !echo">
    <template v-if="echo">
      <CCard :title="t('tools.request-headers.connection')">
        <CKeyValueList :rows="summary" />
      </CCard>
      <NAlert :type="hints.length ? 'warning' : 'success'" class="mb-4" :title="t(hints.length ? 'tools.request-headers.proxyDetected' : 'tools.request-headers.noProxy')" data-testid="hints">
        <ul v-if="hints.length" class="m-0 pl-5">
          <li v-for="h in hints" :key="h">
            {{ t(`tools.request-headers.hints.${h}`) }}
          </li>
        </ul>
      </NAlert>
      <CCard :title="t('tools.request-headers.headers', { count: rows.length })" data-testid="result">
        <ResultTable :columns="columns" :rows="rows" row-key="name" filename="request-headers.json" />
      </CCard>
    </template>
  </NSpin>
</template>
