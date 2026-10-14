<script setup lang="ts">
import type { Accent } from '@/themes'
import { computed } from 'vue'
import { useStyleStore } from '@/stores/style.store'
import { ACCENT_COLORS } from '@/themes'

// A dot in the accent colour for the current mode (decorative; the label carries the meaning).
const props = defineProps<{ accent: Accent }>()
const style = useStyleStore()
const color = computed(() => ACCENT_COLORS[props.accent][style.isDarkTheme ? 'dark' : 'light'].primary)
</script>

<template>
  <span class="inline-block h-3 w-3 shrink-0 rounded-full" :style="{ background: color }" aria-hidden="true" />
</template>
