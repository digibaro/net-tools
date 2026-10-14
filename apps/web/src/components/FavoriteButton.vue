<script setup lang="ts">
import type { Tool } from '@/tools/tool'
import { Star, StarOff } from '@vicons/tabler'
import { NButton, NIcon, NTooltip } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToolStore } from '@/stores/tool.store'

// `tooltip: false` uses a native title instead of a Naive popover (cheaper in long card grids).
const props = withDefaults(defineProps<{ tool: Tool, tooltip?: boolean }>(), { tooltip: true })
const store = useToolStore()
const { t } = useI18n()
const active = computed(() => store.isFavorite(props.tool))
const label = computed(() => t(active.value ? 'favorites.remove' : 'favorites.add'))
</script>

<template>
  <NTooltip v-if="tooltip">
    <template #trigger>
      <NButton
        quaternary circle size="small" :type="active ? 'primary' : 'default'"
        :aria-label="label" :aria-pressed="active" data-testid="favorite-toggle"
        @click.prevent.stop="store.toggleFavorite(tool)"
      >
        <template #icon>
          <NIcon aria-hidden="true" :component="active ? Star : StarOff" />
        </template>
      </NButton>
    </template>
    {{ label }}
  </NTooltip>
  <NButton
    v-else
    quaternary circle size="small" :type="active ? 'primary' : 'default'"
    :aria-label="label" :aria-pressed="active" :title="label" data-testid="favorite-toggle"
    @click.prevent.stop="store.toggleFavorite(tool)"
  >
    <template #icon>
      <NIcon aria-hidden="true" :component="active ? Star : StarOff" />
    </template>
  </NButton>
</template>
