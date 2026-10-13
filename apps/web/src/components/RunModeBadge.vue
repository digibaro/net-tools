<script setup lang="ts">
import type { RunMode } from '@/tools/tool'
import { NTag, NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'

defineProps<{ mode: RunMode, size?: 'tiny' | 'small' | 'medium' }>()
const { t } = useI18n()

const types: Record<RunMode, 'success' | 'info' | 'warning' | 'default'> = {
  'browser': 'success',
  'server': 'info',
  'public-api': 'warning',
  'hybrid': 'default',
}
</script>

<template>
  <NTooltip>
    <template #trigger>
      <NTag :type="types[mode]" :size="size ?? 'small'" :bordered="false" round data-testid="run-mode">
        {{ t(`runMode.${mode}.label`) }}
      </NTag>
    </template>
    {{ t(`runMode.${mode}.description`) }}
  </NTooltip>
</template>
