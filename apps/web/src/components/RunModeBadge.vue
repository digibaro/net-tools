<script setup lang="ts">
import type { RunMode } from '@/tools/tool'
import { NTag, NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'

// `tooltip: false` uses a native title instead of a Naive popover (cheaper in long card grids).
withDefaults(defineProps<{ mode: RunMode, size?: 'tiny' | 'small' | 'medium', tooltip?: boolean }>(), { tooltip: true })
const { t } = useI18n()

const types: Record<RunMode, 'success' | 'info' | 'warning' | 'default'> = {
  'browser': 'success',
  'server': 'info',
  'public-api': 'warning',
  'hybrid': 'default',
}
</script>

<template>
  <NTooltip v-if="tooltip">
    <template #trigger>
      <NTag :type="types[mode]" :size="size ?? 'small'" :bordered="false" round data-testid="run-mode">
        {{ t(`runMode.${mode}.label`) }}
      </NTag>
    </template>
    {{ t(`runMode.${mode}.description`) }}
  </NTooltip>
  <NTag v-else :type="types[mode]" :size="size ?? 'small'" :bordered="false" round :title="t(`runMode.${mode}.description`)" data-testid="run-mode">
    {{ t(`runMode.${mode}.label`) }}
  </NTag>
</template>
