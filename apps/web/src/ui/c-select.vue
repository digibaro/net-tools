<script setup lang="ts" generic="T extends string | number">
import { NFormItem, NSelect } from 'naive-ui'

defineProps<{
  label?: string
  options: { label: string, value: T }[]
  filterable?: boolean
  testId?: string
}>()

const value = defineModel<T>('value', { required: true })
</script>

<template>
  <NFormItem :label="label" :show-label="!!label" :show-feedback="false" class="mb-3">
    <NSelect
      v-model:value="value"
      :options="options"
      :filterable="filterable"
      :input-props="{ 'aria-label': label }"
      :data-testid="testId"
    />
  </NFormItem>
</template>
