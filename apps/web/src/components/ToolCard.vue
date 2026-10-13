<script setup lang="ts">
import type { Tool } from '@/tools/tool'
import { Star, StarOff } from '@vicons/tabler'
import { useThemeVars } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToolStore } from '@/stores/tool.store'

// The home grid renders every tool, so cards use plain elements styled with Naive theme vars
// instead of NCard/NButton/NTag/NTooltip (those cost ~5 component instances per card).
const props = defineProps<{ tool: Tool }>()
const { t } = useI18n()
const theme = useThemeVars()
const store = useToolStore()
const favorite = computed(() => store.isFavorite(props.tool))
const favoriteLabel = computed(() => t(favorite.value ? 'favorites.remove' : 'favorites.add'))
const dot = computed(() => ({
  'browser': theme.value.successColor,
  'server': theme.value.infoColor,
  'public-api': theme.value.warningColor,
  'hybrid': theme.value.textColor3,
})[props.tool.runMode])
</script>

<template>
  <RouterLink
    :to="tool.path"
    class="nt-tool-card box-border block h-full p-3 no-underline transition-shadow"
    :style="{ background: theme.cardColor, border: `1px solid ${theme.dividerColor}`, color: theme.textColor2, borderRadius: theme.borderRadius }"
    :data-testid="`tool-card-${tool.path.slice(1)}`"
  >
    <div class="flex items-start justify-between gap-2">
      <component :is="tool.icon" class="h-8 w-8 opacity-70" aria-hidden="true" />
      <div class="flex items-center gap-1">
        <span v-if="tool.isNew" class="rounded-full px-2 text-xs font-semibold" :style="{ color: theme.primaryColor, border: `1px solid ${theme.primaryColor}` }">
          {{ t('common.new') }}
        </span>
        <button
          type="button"
          class="nt-icon-btn h-7 w-7 flex cursor-pointer items-center justify-center border-0 rounded-full bg-transparent p-0"
          :style="{ color: favorite ? theme.primaryColor : theme.textColor3 }"
          :aria-label="favoriteLabel" :aria-pressed="favorite" :title="favoriteLabel" data-testid="favorite-toggle"
          @click.prevent.stop="store.toggleFavorite(tool)"
        >
          <component :is="favorite ? Star : StarOff" class="h-5 w-5" aria-hidden="true" />
        </button>
      </div>
    </div>
    <h3 class="mb-1 mt-2 text-base font-semibold" :style="{ color: theme.textColor1 }">
      {{ tool.name }}
    </h3>
    <p class="m-0 mb-2 text-sm nt-muted leading-snug">
      {{ tool.description }}
    </p>
    <span class="inline-flex items-center gap-1.5 text-xs nt-muted" :title="t(`runMode.${tool.runMode}.description`)" data-testid="run-mode">
      <span class="inline-block h-2 w-2 rounded-full" :style="{ background: dot }" aria-hidden="true" />
      {{ t(`runMode.${tool.runMode}.label`) }}
    </span>
  </RouterLink>
</template>

<style scoped>
.nt-tool-card:hover,
.nt-tool-card:focus-visible {
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.12);
}
.nt-icon-btn:hover {
  background: rgba(128, 128, 128, 0.15);
}
.nt-icon-btn:focus-visible {
  outline: 2px solid currentColor;
  outline-offset: 1px;
}
</style>
