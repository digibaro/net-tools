<script setup lang="ts">
import { NButton, NFormItem, NInputNumber, NSwitch } from 'naive-ui'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { parseCidr } from '@/utils/ip'
import { generate, MAX_COUNT } from './random-ip-generator.service'

const { t } = useI18n()
const prefix = useQueryParam({ name: 'prefix', defaultValue: '10.0.0.0/8' })
const count = useQueryParam({ name: 'count', defaultValue: 10 })
const unique = useQueryParam({ name: 'unique', defaultValue: true })
const usable = useQueryParam({ name: 'usable', defaultValue: true })
const validation = useValidation({ source: prefix, rules: [{ validator: (v: string) => !!parseCidr(v), message: t('tools.random-ip-generator.invalid') }] })

const result = ref<string[] | string>([])
function regenerate() {
  result.value = generate(prefix.value, { count: count.value, unique: unique.value, usableOnly: usable.value })
}
watch([prefix, count, unique, usable], regenerate, { immediate: true })
const list = computed(() => (Array.isArray(result.value) ? result.value : []))
</script>

<template>
  <CCard>
    <CInputText v-model:value="prefix" :label="t('tools.random-ip-generator.prefix')" placeholder="10.0.0.0/8 or 2001:db8::/32" :validation="validation" monospace test-id="prefix" />
    <div class="flex flex-wrap items-end gap-x-6 gap-y-2">
      <NFormItem :label="t('tools.random-ip-generator.count')" :show-feedback="false">
        <NInputNumber v-model:value="count" :min="1" :max="MAX_COUNT" class="w-32" :input-props="{ 'aria-label': t('tools.random-ip-generator.count') }" data-testid="count" />
      </NFormItem>
      <NFormItem :label="t('tools.random-ip-generator.unique')" :show-feedback="false">
        <NSwitch v-model:value="unique" :aria-label="t('tools.random-ip-generator.unique')" />
      </NFormItem>
      <NFormItem :label="t('tools.random-ip-generator.usable')" :show-feedback="false">
        <NSwitch v-model:value="usable" :aria-label="t('tools.random-ip-generator.usable')" />
      </NFormItem>
      <NButton class="mb-0.5" data-testid="regenerate" @click="regenerate">
        {{ t('tools.random-ip-generator.regenerate') }}
      </NButton>
    </div>
  </CCard>
  <p v-if="result === 'too-small'" class="nt-muted" data-testid="no-result">
    {{ t('tools.random-ip-generator.tooSmall') }}
  </p>
  <CCard v-if="list.length" :title="t('common.result')" data-testid="result">
    <pre class="m-0 max-h-96 overflow-auto text-sm font-mono" tabindex="0">{{ list.join('\n') }}</pre>
    <div class="mt-3">
      <ResultActions :text="() => list.join('\n')" :json="list" filename="random-ips.json" />
    </div>
  </CCard>
</template>
