import type { Tool, ToolCategory } from '@/tools/tool'
import { useStorage } from '@vueuse/core'
import { defineStore } from 'pinia'
import { computed } from 'vue'
import { toolsByCategory } from '@/tools'

export const useToolStore = defineStore('tool', () => {
  const favoritePaths = useStorage<string[]>('nettools:favorites', [])

  const tools = computed<Tool[]>(() => toolsByCategory.flatMap(c => c.components))
  const categories = computed<ToolCategory[]>(() => toolsByCategory)

  const favorites = computed(() => favoritePaths.value
    .map(p => tools.value.find(t => t.path === p))
    .filter((t): t is Tool => !!t))
  const newTools = computed(() => tools.value.filter(t => t.isNew))

  const isFavorite = (tool: Tool) => favoritePaths.value.includes(tool.path)
  function toggleFavorite(tool: Tool) {
    favoritePaths.value = isFavorite(tool)
      ? favoritePaths.value.filter(p => p !== tool.path)
      : [...favoritePaths.value, tool.path]
  }

  return { tools, categories, favorites, newTools, isFavorite, toggleFavorite }
})
