<script setup lang="ts">
import type { Tool } from '@/tools/tool'
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ToolCard from '@/components/ToolCard.vue'
import { useToolStore } from '@/stores/tool.store'

const store = useToolStore()
const { t } = useI18n()

// Render sections progressively (one per idle period) so first load is not one long task.
const visible = ref(2)
let handle: number | undefined
const idle = typeof requestIdleCallback === 'function'
const schedule = (cb: () => void): number => (idle ? requestIdleCallback(cb, { timeout: 200 }) : setTimeout(cb, 16) as unknown as number)
function step() {
  visible.value++
  if (visible.value < sections().length)
    handle = schedule(step)
}
onMounted(() => {
  handle = schedule(step)
})
onUnmounted(() => {
  if (handle !== undefined)
    idle ? cancelIdleCallback(handle) : clearTimeout(handle)
})

function sections(): { id: string, title: string, tools: Tool[] }[] {
  return [
    { id: 'favorites', title: t('home.favorites'), tools: store.favorites },
    { id: 'new', title: t('home.newest'), tools: store.newTools.length === store.tools.length ? [] : store.newTools },
    ...store.categories.map(c => ({ id: `category-${c.id}`, title: c.name, tools: c.components })),
  ].filter(s => s.tools.length > 0)
}
</script>

<template>
  <div class="mx-auto max-w-6xl">
    <h1 class="sr-only">
      {{ t('app.name') }}
    </h1>
    <p v-if="!store.tools.length" class="nt-muted">
      {{ t('home.empty') }}
    </p>
    <section v-for="section in sections().slice(0, visible)" :key="section.id" class="mb-8" :data-testid="`home-${section.id}`">
      <h2 class="mb-3 mt-0 text-lg font-semibold">
        {{ section.title }}
      </h2>
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3 sm:grid-cols-2 xl:grid-cols-4">
        <ToolCard v-for="tool in section.tools" :key="tool.path" :tool="tool" />
      </div>
    </section>
  </div>
</template>
