<script setup lang="ts">
import { NAlert, NButton, NFormItem, NInputNumber, NRadioButton, NRadioGroup } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import RemoteResults from '@/components/RemoteResults.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useRemote } from '@/composables/useRemote'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'
import { hostOf, overall } from './is-it-down.service'

const { t } = useI18n()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const check = useQueryParam({ name: 'check', defaultValue: 'http' })
const locations = useQueryParam({ name: 'locations', defaultValue: '' })
const limit = useQueryParam({ name: 'probes', defaultValue: 6 })
const validation = useValidation({ source: target, rules: [{ validator: (v: string) => isValidHost(hostOf(v)), message: t('tools.ping.invalid') }] })
const remote = useRemote()
const running = computed(() => remote.status.value === 'running')
function run() {
  if (!target.value || !validation.value.isValid)
    return
  remote.start({ type: check.value === 'ping' ? 'ping' : 'http', target: hostOf(target.value), locations: locations.value, limit: limit.value })
}
onMounted(() => {
  if (target.value)
    run()
})
const state = computed(() => (remote.status.value === 'running' ? undefined : overall(remote.meta.value?.type ?? 'http', remote.probes.value)))
const stateType = { up: 'success', partial: 'warning', down: 'error' } as const
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="run">
      <CInputText v-model:value="target" :label="t('tools.is-it-down.target')" placeholder="example.com" :validation="validation" monospace test-id="target" />
      <div class="flex flex-wrap items-end gap-x-6">
        <NFormItem :label="t('tools.is-it-down.check')" :show-feedback="false" class="mb-3">
          <NRadioGroup v-model:value="check" name="check">
            <NRadioButton value="http" label="HTTPS" />
            <NRadioButton value="ping" label="Ping" />
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('remote.limit')" :show-feedback="false" class="mb-3 w-28">
          <NInputNumber v-model:value="limit" :min="1" :max="10" :show-button="false" :input-props="{ 'aria-label': t('remote.limit') }" />
        </NFormItem>
      </div>
      <CInputText v-model:value="locations" :label="t('remote.locations')" :placeholder="t('remote.locationsPlaceholder')" test-id="locations" />
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!target || !validation.isValid || running" data-testid="run">
        {{ t('tools.is-it-down.run') }}
      </NButton>
    </form>
  </CCard>
  <NAlert v-if="state" :type="stateType[state]" class="mb-4" data-testid="overall">
    {{ t(`tools.is-it-down.overall.${state}`, { target: remote.meta.value?.target }) }}
  </NAlert>
  <RemoteResults :meta="remote.meta.value" :probes="remote.probes.value" :summary="remote.summary.value" :status="remote.status.value" :error="remote.error.value" />
</template>
