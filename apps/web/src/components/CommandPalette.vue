<script setup lang="ts">
import type { Component } from 'vue'
import type { ColorMode } from '@/stores/style.store'
import { DeviceDesktop, Home, InfoCircle, Moon, Search, Settings, Sun } from '@vicons/tabler'
import { NIcon, NInput, NModal, useThemeVars } from 'naive-ui'
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useStyleStore } from '@/stores/style.store'
import { useToolStore } from '@/stores/tool.store'
import { createToolSearch } from '@/tools/search'

interface Command {
  id: string
  label: string
  hint?: string
  icon: Component
  run: () => void
}

const show = defineModel<boolean>('show', { default: false })
const { t, locale } = useI18n()
const router = useRouter()
const tools = useToolStore()
const style = useStyleStore()
const themeVars = useThemeVars()
const query = ref('')
const active = ref(0)
const input = ref<InstanceType<typeof NInput>>()
// Rebuilt when the language changes: Fuse indexes the (translated) names once.
const search = computed(() => {
  void locale.value
  return createToolSearch(tools.tools)
})

const staticCommands = computed<Command[]>(() => [
  { id: 'page:home', label: t('nav.home'), icon: Home, run: () => router.push('/') },
  { id: 'page:about', label: t('nav.about'), icon: InfoCircle, run: () => router.push('/about') },
  { id: 'page:settings', label: t('nav.settings'), icon: Settings, run: () => router.push('/settings') },
  ...(['light', 'dark', 'auto'] as ColorMode[]).map(mode => ({
    id: `theme:${mode}`,
    label: t('theme.label', { mode: t(`theme.${mode}`) }),
    icon: { light: Sun, dark: Moon, auto: DeviceDesktop }[mode],
    run: () => {
      style.colorMode = mode
    },
  })),
])

const results = computed<Command[]>(() => {
  const q = query.value.trim().toLowerCase()
  const toolCommands = (q ? search.value(q, 12) : tools.favorites.length ? tools.favorites : tools.tools.slice(0, 8))
    .map(tool => ({ id: tool.path, label: tool.name, hint: tool.description, icon: tool.icon, run: () => router.push(tool.path) }))
  const others = staticCommands.value.filter(c => !q || c.label.toLowerCase().includes(q))
  // A typed page or theme name ("about", "dark") beats fuzzy tool matches.
  return q ? [...others, ...toolCommands] : [...toolCommands, ...others]
})

watch(query, () => {
  active.value = 0
})
watch(show, async (open) => {
  if (open) {
    query.value = ''
    active.value = 0
    await nextTick()
    input.value?.focus()
  }
}, { immediate: true })

function execute(cmd: Command | undefined) {
  if (!cmd)
    return
  show.value = false
  cmd.run()
}

function onKeydown(e: KeyboardEvent) {
  const n = results.value.length
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    active.value = n ? (active.value + 1) % n : 0
  }
  else if (e.key === 'ArrowUp') {
    e.preventDefault()
    active.value = n ? (active.value - 1 + n) % n : 0
  }
  else if (e.key === 'Enter') {
    e.preventDefault()
    execute(results.value[active.value])
  }
}
</script>

<template>
  <NModal v-model:show="show" preset="card" :title="t('search.title')" class="max-w-xl" style="width: calc(100vw - 32px)" :auto-focus="false" data-testid="command-palette">
    <NInput
      ref="input"
      v-model:value="query"
      :placeholder="t('search.placeholder')"
      clearable
      :input-props="{ 'aria-label': t('search.placeholder'), 'aria-controls': 'palette-results', 'role': 'combobox', 'aria-expanded': true }"
      @keydown="onKeydown"
    >
      <template #prefix>
        <NIcon aria-hidden="true" :component="Search" />
      </template>
    </NInput>
    <ul id="palette-results" role="listbox" class="m-0 mt-3 max-h-96 list-none overflow-auto p-0">
      <li
        v-for="(cmd, i) in results"
        :id="`palette-${i}`"
        :key="cmd.id"
        role="option"
        :aria-selected="i === active"
        class="flex cursor-pointer items-center gap-3 rounded px-3 py-2"
        :style="i === active ? { background: themeVars.hoverColor } : undefined"
        @mouseenter="active = i"
        @click="execute(cmd)"
      >
        <NIcon :component="cmd.icon" :size="20" aria-hidden="true" />
        <div class="min-w-0">
          <div class="truncate">
            {{ cmd.label }}
          </div>
          <div v-if="cmd.hint" class="truncate text-xs nt-muted">
            {{ cmd.hint }}
          </div>
        </div>
      </li>
      <li v-if="!results.length" class="px-3 py-2 nt-muted">
        {{ t('search.noResults') }}
      </li>
    </ul>
  </NModal>
</template>
