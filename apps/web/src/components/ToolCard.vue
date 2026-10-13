<script setup lang="ts">
import type { Tool } from '@/tools/tool'
import { NCard, NIcon, NTag } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import FavoriteButton from './FavoriteButton.vue'
import RunModeBadge from './RunModeBadge.vue'

defineProps<{ tool: Tool }>()
const { t } = useI18n()
</script>

<template>
  <RouterLink :to="tool.path" class="block h-full rounded no-underline" :data-testid="`tool-card-${tool.path.slice(1)}`">
    <NCard hoverable size="small" class="h-full">
      <div class="flex items-start justify-between gap-2">
        <NIcon :component="tool.icon" :size="32" class="opacity-70" aria-hidden="true" />
        <div class="flex items-center gap-1">
          <NTag v-if="tool.isNew" type="success" size="tiny" round :bordered="false">
            {{ t('common.new') }}
          </NTag>
          <FavoriteButton :tool="tool" />
        </div>
      </div>
      <h3 class="mb-1 mt-2 text-base font-semibold">
        {{ tool.name }}
      </h3>
      <p class="m-0 mb-2 text-sm nt-muted leading-snug">
        {{ tool.description }}
      </p>
      <RunModeBadge :mode="tool.runMode" size="tiny" />
    </NCard>
  </RouterLink>
</template>
