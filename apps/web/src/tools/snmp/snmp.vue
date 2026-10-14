<script setup lang="ts">
import type { SnmpRequest, SnmpResult } from './snmp.service'
import type { ApiError as ApiErrorType } from '@/api/client'
import { NAlert, NButton, NFormItem, NInput, NInputNumber, NRadioButton, NRadioGroup, NSelect, NTag } from 'naive-ui'
import { computed, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ApiError as ApiErrorClass, apiPost } from '@/api/client'
import ApiError from '@/components/ApiError.vue'
import ResultActions from '@/components/ResultActions.vue'
import ResultTable from '@/components/ResultTable.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { isValidHost } from '@/utils/host'
import { displayValue, isOidList, oidName, PRESETS, splitOids } from './snmp.service'

const { t } = useI18n()
const target = useQueryParam({ name: 'target', defaultValue: '' })
const port = useQueryParam({ name: 'port', defaultValue: 161 })
const version = useQueryParam({ name: 'version', defaultValue: '2c' })
const operation = useQueryParam({ name: 'op', defaultValue: 'walk' })
const oids = useQueryParam({ name: 'oid', defaultValue: '1.3.6.1.2.1.1' })
const maxRows = useQueryParam({ name: 'maxRows', defaultValue: 200 })
// Secrets stay out of the URL.
const community = ref('public')
const user = ref('')
const level = ref<'noAuthNoPriv' | 'authNoPriv' | 'authPriv'>('authPriv')
const authProtocol = ref<'MD5' | 'SHA' | 'SHA224' | 'SHA256' | 'SHA384' | 'SHA512'>('SHA256')
const authPassword = ref('')
const privProtocol = ref<'DES' | 'AES' | 'AES192' | 'AES256' | 'AES192C' | 'AES256C'>('AES')
const privPassword = ref('')

const targetV = useValidation({ source: target, rules: [{ validator: isValidHost, message: t('tools.ping.invalid') }] })
const oidV = useValidation({ source: oids, rules: [
  { validator: isOidList, message: t('tools.snmp.invalidOid') },
  { validator: (s: string) => operation.value === 'get' || splitOids(s).length === 1, message: t('tools.snmp.oneOid') },
] })
const result = shallowRef<SnmpResult>()
const error = ref<ApiErrorType>()
const running = ref(false)
const canRun = computed(() => !!target.value && targetV.value.isValid && oidV.value.isValid && !!oids.value.trim())

async function run() {
  if (!canRun.value)
    return
  running.value = true
  error.value = undefined
  result.value = undefined
  const body: SnmpRequest = { target: target.value.trim(), port: port.value, version: version.value as '2c' | '3', operation: operation.value as 'get' | 'walk', oids: splitOids(oids.value), maxRows: maxRows.value }
  if (version.value === '3')
    body.v3 = { user: user.value, level: level.value, authProtocol: authProtocol.value, authPassword: authPassword.value || undefined, privProtocol: privProtocol.value, privPassword: privPassword.value || undefined }
  else
    body.community = community.value
  try {
    result.value = await apiPost<SnmpResult>('/v1/snmp', body)
  }
  catch (e) {
    error.value = e instanceof ApiErrorClass ? e : new ApiErrorClass('INTERNAL', String(e))
  }
  finally {
    running.value = false
  }
}
const opt = (xs: readonly string[]) => xs.map(x => ({ label: x, value: x }))
const rows = computed(() => (result.value?.vars ?? []).map(v => ({ oid: v.oid, name: oidName(v.oid), type: v.type, value: displayValue(v) })))
const c = (k: string) => t(`tools.snmp.columns.${k}`)
</script>

<template>
  <CCard>
    <form class="contents" autocomplete="off" @submit.prevent="run">
      <div class="grid gap-x-4 sm:grid-cols-[1fr_7rem]">
        <CInputText v-model:value="target" :label="t('tools.snmp.target')" placeholder="192.168.1.1" :validation="targetV" monospace test-id="target" />
        <NFormItem :label="t('tools.snmp.port')" :show-feedback="false" class="mb-3">
          <NInputNumber v-model:value="port" :min="1" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.snmp.port') }" />
        </NFormItem>
      </div>
      <div class="flex flex-wrap gap-x-6">
        <NFormItem :label="t('tools.snmp.version')" :show-feedback="false" class="mb-3">
          <NRadioGroup v-model:value="version" name="snmp-version">
            <NRadioButton value="2c" label="v2c" />
            <NRadioButton value="3" label="v3" />
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('tools.snmp.operation')" :show-feedback="false" class="mb-3">
          <NRadioGroup v-model:value="operation" name="snmp-op">
            <NRadioButton value="walk" :label="t('tools.snmp.walk')" />
            <NRadioButton value="get" :label="t('tools.snmp.get')" />
          </NRadioGroup>
        </NFormItem>
        <NFormItem v-if="operation === 'walk'" :label="t('tools.snmp.maxRows')" :show-feedback="false" class="mb-3 w-28">
          <NInputNumber v-model:value="maxRows" :min="1" :max="1000" :show-button="false" :input-props="{ 'aria-label': t('tools.snmp.maxRows') }" />
        </NFormItem>
      </div>
      <NFormItem v-if="version === '2c'" :label="t('tools.snmp.community')" :show-feedback="false" class="mb-3 max-w-80">
        <NInput v-model:value="community" type="password" show-password-on="click" :input-props="{ 'aria-label': t('tools.snmp.community'), 'autocomplete': 'off' }" data-testid="community" />
      </NFormItem>
      <div v-else class="grid gap-x-4 sm:grid-cols-3">
        <NFormItem :label="t('tools.snmp.user')" :show-feedback="false" class="mb-3">
          <NInput v-model:value="user" :input-props="{ 'aria-label': t('tools.snmp.user'), 'autocomplete': 'off' }" />
        </NFormItem>
        <NFormItem :label="t('tools.snmp.level')" :show-feedback="false" class="mb-3 sm:col-span-2">
          <NRadioGroup v-model:value="level" name="snmp-level">
            <NRadioButton value="noAuthNoPriv" label="noAuthNoPriv" />
            <NRadioButton value="authNoPriv" label="authNoPriv" />
            <NRadioButton value="authPriv" label="authPriv" />
          </NRadioGroup>
        </NFormItem>
        <template v-if="level !== 'noAuthNoPriv'">
          <NFormItem :label="t('tools.snmp.authProtocol')" :show-feedback="false" class="mb-3">
            <NSelect v-model:value="authProtocol" :options="opt(['MD5', 'SHA', 'SHA224', 'SHA256', 'SHA384', 'SHA512'])" :aria-label="t('tools.snmp.authProtocol')" />
          </NFormItem>
          <NFormItem :label="t('tools.snmp.authPassword')" :show-feedback="false" class="mb-3 sm:col-span-2">
            <NInput v-model:value="authPassword" type="password" show-password-on="click" :input-props="{ 'aria-label': t('tools.snmp.authPassword'), 'autocomplete': 'off' }" />
          </NFormItem>
        </template>
        <template v-if="level === 'authPriv'">
          <NFormItem :label="t('tools.snmp.privProtocol')" :show-feedback="false" class="mb-3">
            <NSelect v-model:value="privProtocol" :options="opt(['DES', 'AES', 'AES192', 'AES256', 'AES192C', 'AES256C'])" :aria-label="t('tools.snmp.privProtocol')" />
          </NFormItem>
          <NFormItem :label="t('tools.snmp.privPassword')" :show-feedback="false" class="mb-3 sm:col-span-2">
            <NInput v-model:value="privPassword" type="password" show-password-on="click" :input-props="{ 'aria-label': t('tools.snmp.privPassword'), 'autocomplete': 'off' }" />
          </NFormItem>
        </template>
      </div>
      <CInputText v-model:value="oids" :label="operation === 'walk' ? t('tools.snmp.oidWalk') : t('tools.snmp.oidGet')" placeholder="1.3.6.1.2.1.1" :validation="oidV" monospace test-id="oid" />
      <div class="mb-3 flex flex-wrap gap-1">
        <NTag v-for="p in PRESETS" :key="p.key" size="small" class="cursor-pointer" :bordered="false" :type="oids === p.oid ? 'primary' : 'default'" @click="oids = p.oid; operation = 'walk'">
          {{ t(`tools.snmp.presets.${p.key}`) }}
        </NTag>
      </div>
      <NButton type="primary" attr-type="submit" :loading="running" :disabled="!canRun || running" data-testid="run">
        {{ t('tools.snmp.run') }}
      </NButton>
      <p class="mb-0 text-xs nt-muted">
        {{ t('tools.snmp.note') }}
      </p>
    </form>
  </CCard>
  <ApiError v-if="error" :error="error" />
  <CCard v-if="result" :title="t('tools.snmp.found', { n: result.vars.length })" data-testid="result">
    <NAlert v-if="result.truncated" type="info" class="mb-3">
      {{ t('tools.snmp.truncated', { n: result.vars.length }) }}
    </NAlert>
    <ResultTable :columns="[{ key: 'name', title: c('name'), mono: true, width: 220 }, { key: 'type', title: c('type'), width: 120 }, { key: 'value', title: c('value'), mono: true, width: 360 }, { key: 'oid', title: 'OID', mono: true, width: 260 }]" :rows="rows" row-key="oid" :actions="false" />
    <div class="mt-3">
      <ResultActions :text="() => result!.vars.map(v => `${v.oid} = ${v.type}: ${v.value}`).join('\n')" :json="result" filename="snmp.json" />
    </div>
  </CCard>
</template>
