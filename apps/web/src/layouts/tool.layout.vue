<script setup lang="ts">
import type { Tool } from '@/tools/tool'
import { Printer } from '@vicons/tabler'
import { NButton, NIcon, NTooltip } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import FavoriteButton from '@/components/FavoriteButton.vue'
import RunModeBadge from '@/components/RunModeBadge.vue'
import ToolDisabled from '@/components/ToolDisabled.vue'
import { useCapabilitiesStore } from '@/stores/capabilities.store'

const props = defineProps<{ tool: Tool }>()
const caps = useCapabilitiesStore()
const reason = computed(() => caps.reasonFor(props.tool))
const { t } = useI18n()
// The browser's print dialog also saves as PDF; the print stylesheet in App.vue hides the chrome.
const print = () => window.print()
</script>

<template>
  <div class="mx-auto max-w-4xl">
    <header class="mb-5">
      <div class="flex flex-wrap items-center gap-2">
        <NIcon :component="tool.icon" :size="28" aria-hidden="true" />
        <h1 class="m-0 text-2xl font-semibold">
          {{ tool.name }}
        </h1>
        <RunModeBadge :mode="tool.runMode" />
        <FavoriteButton :tool="tool" />
        <NTooltip>
          <template #trigger>
            <NButton quaternary circle size="small" class="nt-no-print" :aria-label="t('common.print')" data-testid="print" @click="print()">
              <template #icon>
                <NIcon aria-hidden="true" :component="Printer" />
              </template>
            </NButton>
          </template>
          {{ t('common.print') }}
        </NTooltip>
      </div>
      <p class="mb-0 mt-2 nt-muted">
        {{ tool.description }}
      </p>
    </header>
    <ToolDisabled v-if="reason" :reason="reason" />
    <slot />
  </div>
</template>
