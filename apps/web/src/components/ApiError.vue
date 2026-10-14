<script setup lang="ts">
import type { ApiError } from '@/api/client'
import { NAlert } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ error: ApiError }>()
const { t, te } = useI18n()

const title = computed(() => {
  const key = `apiErrors.${props.error.code}`
  return te(key) ? t(key) : t('apiErrors.INTERNAL')
})
</script>

<template>
  <NAlert type="error" :title="title" class="mb-4" data-testid="api-error">
    <span>{{ error.message }}</span>
    <code class="ml-2 text-xs opacity-70">{{ error.code }}</code>
  </NAlert>
</template>
