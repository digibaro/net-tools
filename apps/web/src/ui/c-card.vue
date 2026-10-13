<script setup lang="ts">
import { NCard } from 'naive-ui'

defineProps<{ title?: string }>()
</script>

<template>
  <NCard :title="title" size="small" class="mb-4" :segmented="title ? { content: true } : undefined">
    <template v-if="$slots['header-extra']" #header-extra>
      <slot name="header-extra" />
    </template>
    <slot />
  </NCard>
</template>
