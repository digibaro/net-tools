<script setup lang="ts">
import type { InputHTMLAttributes } from 'vue'
import type { Validation } from '@/composables/useValidation'
import { NFormItem, NInput } from 'naive-ui'
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  label?: string
  placeholder?: string
  validation?: Validation
  multiline?: boolean
  rows?: number
  monospace?: boolean
  clearable?: boolean
  readonly?: boolean
  autofocus?: boolean
  testId?: string
}>(), { rows: 4, clearable: true })

const value = defineModel<string>('value', { default: '' })

const inputProps = computed(() => ({
  'spellcheck': false,
  'autocomplete': 'off',
  'autocapitalize': 'off',
  'aria-label': props.label ?? props.placeholder,
  'data-testid': props.testId,
}) as InputHTMLAttributes)
</script>

<template>
  <NFormItem
    :label="label"
    :show-label="!!label"
    :feedback="props.validation?.message"
    :validation-status="props.validation?.status"
    :show-feedback="!!props.validation?.message"
    class="mb-1"
  >
    <NInput
      v-model:value="value"
      :type="multiline ? 'textarea' : 'text'"
      :rows="rows"
      :placeholder="placeholder"
      :clearable="clearable && !readonly"
      :readonly="readonly"
      :autofocus="autofocus"
      :input-props="inputProps"
      :class="{ 'font-mono': monospace }"
    />
  </NFormItem>
</template>
