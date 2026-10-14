<script setup lang="ts">
import { NFormItem, NInputNumber, NRadioButton, NRadioGroup } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import CInputText from '@/ui/c-input-text.vue'

/** Chooses where a measurement runs: this server, or Globalping probes worldwide. */
defineProps<{ available: boolean }>()
const vantage = defineModel<string>('vantage', { required: true })
const locations = defineModel<string>('locations', { required: true })
const limit = defineModel<number>('limit', { required: true })
const { t } = useI18n()
</script>

<template>
  <div v-if="available" class="mb-1" data-testid="vantage">
    <NFormItem :label="t('remote.vantage')" :show-feedback="false" class="mb-3">
      <NRadioGroup v-model:value="vantage" name="vantage">
        <NRadioButton value="server" :label="t('remote.server')" />
        <NRadioButton value="worldwide" :label="t('remote.worldwide')" data-testid="vantage-worldwide" />
      </NRadioGroup>
    </NFormItem>
    <div v-if="vantage === 'worldwide'" class="grid gap-x-4 sm:grid-cols-[1fr_8rem]">
      <CInputText v-model:value="locations" :label="t('remote.locations')" :placeholder="t('remote.locationsPlaceholder')" test-id="locations" />
      <NFormItem :label="t('remote.limit')" :show-feedback="false" class="mb-3">
        <NInputNumber v-model:value="limit" :min="1" :max="10" :show-button="false" :input-props="{ 'aria-label': t('remote.limit') }" />
      </NFormItem>
    </div>
  </div>
</template>
