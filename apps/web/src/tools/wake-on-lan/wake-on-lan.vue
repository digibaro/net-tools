<script setup lang="ts">
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NFormItem, NRadioButton, NRadioGroup } from 'naive-ui'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiPost } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isBroadcast, normalizeMac } from './wake-on-lan.service'

const { t } = useI18n()
const mac = useQueryParam({ name: 'mac', defaultValue: '' })
const broadcast = useQueryParam({ name: 'broadcast', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 9 })
const macV = useValidation({ source: mac, rules: [{ validator: (v: string) => !!normalizeMac(v), message: t('tools.wake-on-lan.invalidMac') }] })
const bcV = useValidation({ source: broadcast, rules: [{ validator: isBroadcast, message: t('tools.wake-on-lan.invalidBroadcast') }] })
const sending = ref(false)
const sent = ref('')
const error = ref<ApiErrorType>()
async function send() {
  const m = normalizeMac(mac.value)
  if (!m || !bcV.value.isValid)
    return
  sending.value = true
  error.value = undefined
  sent.value = ''
  try {
    const r = await apiPost<{ destination: string }>('/v1/lan/wol', { mac: m, broadcast: broadcast.value.trim() || undefined, port: port.value })
    sent.value = t('tools.wake-on-lan.sent', { mac: m, to: r.destination })
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    sending.value = false
  }
}
</script>

<template>
  <CCard>
    <form class="contents" @submit.prevent="send">
      <div class="grid gap-x-4 sm:grid-cols-2">
        <CInputText v-model:value="mac" :label="t('tools.wake-on-lan.mac')" placeholder="00:11:22:33:44:55" :validation="macV" monospace test-id="mac" />
        <CInputText v-model:value="broadcast" :label="t('tools.wake-on-lan.broadcast')" placeholder="255.255.255.255" :validation="bcV" monospace test-id="broadcast" />
      </div>
      <NFormItem :label="t('tools.wake-on-lan.port')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="port" name="wol-port">
          <NRadioButton :value="9" label="9" />
          <NRadioButton :value="7" label="7" />
        </NRadioGroup>
      </NFormItem>
      <NButton type="primary" attr-type="submit" :loading="sending" :disabled="!mac || !macV.isValid || !bcV.isValid" data-testid="run">
        {{ t('tools.wake-on-lan.send') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.wake-on-lan.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <NAlert v-if="sent" type="success" data-testid="result">
    {{ sent }}
  </NAlert>
</template>
