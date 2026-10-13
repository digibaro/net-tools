<script setup lang="ts">
import { NCard } from 'naive-ui'

// Title is rendered as a real <h2> (Naive's card header uses role="heading" without a level).
defineProps<{ title?: string }>()
</script>

<template>
  <NCard size="small" class="mb-4">
    <div v-if="title || $slots['header-extra']" class="mb-3 flex items-center justify-between gap-2">
      <h2 v-if="title" class="m-0 text-base font-semibold">
        {{ title }}
      </h2>
      <slot name="header-extra" />
    </div>
    <slot />
  </NCard>
</template>
