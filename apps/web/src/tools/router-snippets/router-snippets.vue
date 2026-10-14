<script setup lang="ts">
import type { Ref } from 'vue'
import type { Action, Kind, Platform, PrefixEntry } from './router-snippets.service'
import { NFormItem, NInputNumber, NRadioButton, NRadioGroup, NSwitch } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CSelect from '@/ui/c-select.vue'
import { aggregateCidrs, cidrToString } from '@/utils/ip'
import { acl, isListName, parsePrefixes, PLATFORMS, prefixList, withUpTo } from './router-snippets.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'list', defaultValue: '192.0.2.0/24\n198.51.100.0/24\n203.0.113.0/25\n2001:db8::/32' })
const kind = useQueryParam({ name: 'kind', defaultValue: 'prefix-list' }) as Ref<Kind>
const platform = useQueryParam({ name: 'platform', defaultValue: 'ios' }) as Ref<Platform>
const name = useQueryParam({ name: 'name', defaultValue: 'CUSTOMER-IN' })
const action = useQueryParam({ name: 'action', defaultValue: 'permit' }) as Ref<Action>
const aggregate = useQueryParam({ name: 'aggregate', defaultValue: false })
const upTo4 = useQueryParam({ name: 'upto4', defaultValue: 0 })
const upTo6 = useQueryParam({ name: 'upto6', defaultValue: 0 })

const parsed = computed(() => parsePrefixes(input.value))
const inputV = computed(() => parsed.value.invalid.length
  ? { isValid: false, status: 'error' as const, message: t('tools.router-snippets.invalid', { entries: parsed.value.invalid.slice(0, 5).join(', '), count: parsed.value.invalid.length }) }
  : undefined)
const nameV = useValidation({ source: name, rules: [{ validator: isListName, message: t('tools.router-snippets.invalidName') }] })
const entries = computed<PrefixEntry[]>(() => {
  const es = aggregate.value ? aggregateCidrs(parsed.value.entries.map(e => e.cidr)).map(cidr => ({ cidr })) : parsed.value.entries
  return kind.value === 'prefix-list' ? withUpTo(es, upTo4.value || undefined, upTo6.value || undefined) : es
})
const output = computed(() => {
  if (!entries.value.length || !nameV.value.isValid || !name.value)
    return ''
  return kind.value === 'acl' ? acl(entries.value, platform.value, name.value, action.value) : prefixList(entries.value, platform.value, name.value, action.value)
})
const platformOptions = computed(() => PLATFORMS.map(p => ({ label: t(`tools.router-snippets.platforms.${p}`), value: p })))
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.router-snippets.input')" :validation="inputV" multiline :rows="8" monospace test-id="input" />
    <div class="grid gap-x-4 sm:grid-cols-2">
      <CSelect v-model:value="platform" :label="t('tools.router-snippets.platform')" :options="platformOptions" test-id="platform" />
      <CInputText v-model:value="name" :label="t('tools.router-snippets.name')" :validation="nameV" monospace test-id="name" />
    </div>
    <div class="flex flex-wrap items-end gap-x-6">
      <NFormItem :label="t('tools.router-snippets.kind')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="kind" name="snippet-kind">
          <NRadioButton value="prefix-list" :label="t('tools.router-snippets.kinds.prefixList')" />
          <NRadioButton value="acl" :label="t('tools.router-snippets.kinds.acl')" />
        </NRadioGroup>
      </NFormItem>
      <NFormItem :label="t('tools.router-snippets.action')" :show-feedback="false" class="mb-3">
        <NRadioGroup v-model:value="action" name="snippet-action">
          <NRadioButton value="permit" label="permit" />
          <NRadioButton value="deny" label="deny" />
        </NRadioGroup>
      </NFormItem>
      <NFormItem :label="t('tools.router-snippets.aggregate')" :show-feedback="false" class="mb-3">
        <NSwitch v-model:value="aggregate" :aria-label="t('tools.router-snippets.aggregate')" />
      </NFormItem>
      <template v-if="kind === 'prefix-list'">
        <NFormItem :label="t('tools.router-snippets.upTo4')" :show-feedback="false" class="mb-3 w-36">
          <NInputNumber v-model:value="upTo4" :min="0" :max="32" :show-button="false" :input-props="{ 'aria-label': t('tools.router-snippets.upTo4') }" />
        </NFormItem>
        <NFormItem :label="t('tools.router-snippets.upTo6')" :show-feedback="false" class="mb-3 w-36">
          <NInputNumber v-model:value="upTo6" :min="0" :max="128" :show-button="false" :input-props="{ 'aria-label': t('tools.router-snippets.upTo6') }" />
        </NFormItem>
      </template>
    </div>
    <p class="mb-0 text-xs nt-muted">
      {{ t('tools.router-snippets.note') }}
    </p>
  </CCard>
  <CCard v-if="output" :title="t('tools.router-snippets.output', { n: entries.length })" data-testid="result">
    <pre class="m-0 mb-4 overflow-auto text-sm font-mono" data-testid="output">{{ output }}</pre>
    <ResultActions :text="output" :json="{ platform, kind, name, entries: entries.map(e => ({ prefix: cidrToString(e.cidr), ge: e.ge, le: e.le })) }" filename="router-snippet.json" />
  </CCard>
</template>
