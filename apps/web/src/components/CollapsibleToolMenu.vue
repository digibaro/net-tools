<script setup lang="ts">
import type { MenuOption } from 'naive-ui'
import { useStorage } from '@vueuse/core'
import { NIcon, NMenu } from 'naive-ui'
import { computed, h } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useToolStore } from '@/stores/tool.store'

const emit = defineEmits<{ navigate: [] }>()
const store = useToolStore()
const route = useRoute()
const collapsedCategories = useStorage<string[]>('nettools:collapsed-categories', [])

const options = computed<MenuOption[]>(() => store.categories.map(c => ({
  key: `category:${c.id}`,
  label: c.name,
  children: c.components.map(tool => ({
    key: tool.path,
    label: () => h(RouterLink, { to: tool.path, onClick: () => emit('navigate') }, { default: () => tool.name }),
    icon: () => h(NIcon, { 'component': tool.icon, 'aria-hidden': 'true' }),
  })),
})))

const expandedKeys = computed(() => store.categories
  .map(c => `category:${c.id}`)
  .filter(k => !collapsedCategories.value.includes(k)))

function onExpand(keys: string[]) {
  collapsedCategories.value = store.categories
    .map(c => `category:${c.id}`)
    .filter(k => !keys.includes(k))
}
</script>

<template>
  <NMenu
    :options="options"
    :value="route.path"
    :expanded-keys="expandedKeys"
    :indent="16"
    :root-indent="12"
    @update:expanded-keys="onExpand"
  />
</template>
