<script setup lang="ts">
import { NAlert } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ reason: string }>()
const { t, te } = useI18n()
const text = computed(() => {
  const key = `capabilities.reasons.${props.reason}`
  return te(key) ? t(key) : t('capabilities.reasons.unavailable')
})
</script>

<template>
  <NAlert type="warning" :title="t('capabilities.disabledTitle')" class="mb-4" data-testid="tool-disabled">
    {{ text }}
  </NAlert>
</template>
