<script setup lang="ts">
import { InfoCircle, LayoutSidebarLeftCollapse, Menu2, Search, Settings } from '@vicons/tabler'
import { onKeyStroke } from '@vueuse/core'
import { NButton, NIcon, NLayout, NLayoutContent, NLayoutHeader, NLayoutSider, NTooltip } from 'naive-ui'
import { computed, defineAsyncComponent, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import CollapsibleToolMenu from '@/components/CollapsibleToolMenu.vue'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import { useStyleStore } from '@/stores/style.store'

const style = useStyleStore()
const { t } = useI18n()
const router = useRouter()
// The palette (fuse.js, modal, input) is loaded on first use to keep the initial bundle small.
const CommandPalette = defineAsyncComponent(() => import('@/components/CommandPalette.vue'))
const paletteOpen = ref(false)
const paletteLoaded = ref(false)
function openPalette(open = true) {
  paletteLoaded.value = true
  paletteOpen.value = open
}
onKeyStroke(['k', 'K'], (e) => {
  if (e.ctrlKey || e.metaKey) {
    e.preventDefault()
    openPalette(!paletteOpen.value)
  }
})
const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
const shortcut = isMac ? '⌘K' : 'Ctrl+K'

const mobileOpen = ref(false)
const collapsed = computed({
  get: () => style.isSmallScreen ? !mobileOpen.value : style.isMenuCollapsed,
  set: (v: boolean) => {
    if (style.isSmallScreen)
      mobileOpen.value = !v
    else
      style.isMenuCollapsed = v
  },
})

function closeOnMobile() {
  if (style.isSmallScreen)
    mobileOpen.value = false
}
</script>

<template>
  <NLayout has-sider class="h-dvh" :position="style.isSmallScreen ? 'absolute' : 'static'">
    <NLayoutSider
      v-model:collapsed="collapsed"
      bordered
      collapse-mode="transform"
      :collapsed-width="0"
      :width="260"
      :native-scrollbar="false"
      :position="style.isSmallScreen ? 'absolute' : 'static'"
      :class="{ 'z-20': style.isSmallScreen }"
      content-class="flex flex-col"
      :aria-label="t('nav.tools')"
      data-testid="sidebar"
      :inert="collapsed || undefined"
    >
      <RouterLink to="/" class="flex items-center gap-2 px-4 py-4 text-lg font-semibold no-underline" style="color: inherit" @click="closeOnMobile">
        <img src="/favicon.svg" alt="" width="28" height="28">
        {{ t('app.name') }}
      </RouterLink>
      <!-- On small screens the collapsed sidebar is off-canvas; mount the menu only when opened. -->
      <CollapsibleToolMenu v-if="!style.isSmallScreen || mobileOpen" @navigate="closeOnMobile" />
    </NLayoutSider>
    <div v-if="style.isSmallScreen && mobileOpen" class="fixed inset-0 z-10 bg-black/40" aria-hidden="true" @click="mobileOpen = false" />
    <NLayout :native-scrollbar="false">
      <NLayoutHeader bordered class="sticky top-0 z-5 flex items-center gap-2 px-3 py-2">
        <NTooltip>
          <template #trigger>
            <NButton quaternary circle :aria-label="t('nav.toggleMenu')" data-testid="menu-toggle" @click="collapsed = !collapsed">
              <template #icon>
                <NIcon aria-hidden="true" :component="collapsed ? Menu2 : LayoutSidebarLeftCollapse" />
              </template>
            </NButton>
          </template>
          {{ t('nav.toggleMenu') }}
        </NTooltip>
        <NButton class="max-w-md flex-1 justify-start" secondary data-testid="search-open" @click="openPalette()">
          <template #icon>
            <NIcon aria-hidden="true" :component="Search" />
          </template>
          <span class="flex-1 text-left nt-muted">{{ t('search.placeholder') }}</span>
          <kbd class="ml-2 hidden text-xs nt-muted sm:inline">{{ shortcut }}</kbd>
        </NButton>
        <div class="ml-auto flex items-center gap-1">
          <ThemeSwitcher />
          <NTooltip>
            <template #trigger>
              <NButton quaternary circle :aria-label="t('nav.settings')" @click="router.push('/settings')">
                <template #icon>
                  <NIcon aria-hidden="true" :component="Settings" />
                </template>
              </NButton>
            </template>
            {{ t('nav.settings') }}
          </NTooltip>
          <NTooltip>
            <template #trigger>
              <NButton quaternary circle :aria-label="t('nav.about')" @click="router.push('/about')">
                <template #icon>
                  <NIcon aria-hidden="true" :component="InfoCircle" />
                </template>
              </NButton>
            </template>
            {{ t('nav.about') }}
          </NTooltip>
        </div>
      </NLayoutHeader>
      <NLayoutContent class="px-4 py-6 sm:px-6">
        <main>
          <slot />
        </main>
      </NLayoutContent>
    </NLayout>
    <CommandPalette v-if="paletteLoaded" v-model:show="paletteOpen" />
  </NLayout>
</template>
