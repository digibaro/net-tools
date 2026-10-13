<script setup lang="ts">
import type { ApiError as ApiErrorType, EchoInfo } from '@/api/client'
import { Refresh } from '@vicons/tabler'
import { NAlert, NButton, NIcon, NSpin } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getEcho } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { fetchPublicIp, relation } from './what-is-my-ip.service'

const { t } = useI18n()
const loading = ref(false)
const echo = shallowRef<EchoInfo>()
const echoError = shallowRef<ApiErrorType>()
const publicV4 = ref<string>()
const publicV6 = ref<string>()

async function refresh() {
  loading.value = true
  echoError.value = undefined
  const [e, v4, v6] = await Promise.allSettled([getEcho(), fetchPublicIp(4), fetchPublicIp(6)])
  echo.value = e.status === 'fulfilled' ? e.value : undefined
  echoError.value = e.status === 'rejected' ? e.reason as ApiErrorType : undefined
  publicV4.value = v4.status === 'fulfilled' ? v4.value : undefined
  publicV6.value = v6.status === 'fulfilled' ? v6.value : undefined
  loading.value = false
}
onMounted(refresh)

const notAvailable = computed(() => t('tools.what-is-my-ip.notAvailable'))
const publicRows = computed(() => [
  { label: t('tools.what-is-my-ip.ipv4'), value: publicV4.value ?? notAvailable.value, mono: !!publicV4.value, noCopy: !publicV4.value },
  { label: t('tools.what-is-my-ip.ipv6'), value: publicV6.value ?? notAvailable.value, mono: !!publicV6.value, noCopy: !publicV6.value },
])
const serverRows = computed(() => echo.value
  ? [
      { label: t('tools.what-is-my-ip.address'), value: echo.value.ip, mono: true },
      { label: t('tools.what-is-my-ip.family'), value: `IPv${echo.value.family}`, noCopy: true },
      { label: t('tools.what-is-my-ip.protocol'), value: `${echo.value.protocol} (${echo.value.scheme})`, noCopy: true },
    ]
  : [])
const rel = computed(() => (echo.value && (publicV4.value || publicV6.value)) ? relation(echo.value.ip, [publicV4.value, publicV6.value]) : undefined)
const json = computed(() => ({ public: { ipv4: publicV4.value ?? null, ipv6: publicV6.value ?? null }, server: echo.value ?? null }))
const asText = () => [...publicRows.value, ...serverRows.value].map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <div class="mb-4 flex justify-end">
    <NButton :loading="loading" data-testid="refresh" @click="refresh">
      <template #icon>
        <NIcon aria-hidden="true" :component="Refresh" />
      </template>
      {{ t('tools.what-is-my-ip.refresh') }}
    </NButton>
  </div>
  <NSpin :show="loading && !echo && !echoError">
    <CCard :title="t('tools.what-is-my-ip.publicTitle')" data-testid="public">
      <CKeyValueList :rows="publicRows" />
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.what-is-my-ip.ipifyNote') }}
      </p>
    </CCard>
    <CCard :title="t('tools.what-is-my-ip.serverTitle')" data-testid="server">
      <ApiError v-if="echoError" :error="echoError" />
      <CKeyValueList v-else :rows="serverRows" />
      <NAlert v-if="rel && rel !== 'same'" type="info" class="mt-3" data-testid="relation">
        {{ t(`tools.what-is-my-ip.relation.${rel}`) }}
      </NAlert>
    </CCard>
  </NSpin>
  <ResultActions v-if="echo || publicV4 || publicV6" :text="asText" :json="json" filename="my-ip.json" />
</template>
