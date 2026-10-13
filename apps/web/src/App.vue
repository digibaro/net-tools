<script setup lang="ts">
import { darkTheme, NConfigProvider, NGlobalStyle, useOsTheme } from 'naive-ui'
import { computed } from 'vue'

// Phase 1 replaces this with the light/dark/auto style store (docs/THEMING.md).
const osTheme = useOsTheme()
const theme = computed(() => (osTheme.value === 'dark' ? darkTheme : null))
</script>

<template>
  <NConfigProvider :theme="theme">
    <NGlobalStyle />
    <RouterView />
  </NConfigProvider>
</template>
