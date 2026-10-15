import type { Accent } from '@/themes'
import { useMediaQuery, useStorage } from '@vueuse/core'
import { defineStore } from 'pinia'
import { computed, watchEffect } from 'vue'
import { ACCENT_COLORS, ACCENTS, DEFAULT_ACCENT, faviconDataUrl, THEME_COLOR } from '@/themes'

export type ColorMode = 'light' | 'dark' | 'auto'
export const COLOR_MODES: ColorMode[] = ['light', 'dark', 'auto']
/** Read by the anti-FOUC script in index.html; keep both in sync. */
export const COLOR_MODE_STORAGE_KEY = 'nettools:color-mode'

export function resolveDark(mode: ColorMode, prefersDark: boolean): boolean {
  return mode === 'auto' ? prefersDark : mode === 'dark'
}

export function nextColorMode(mode: ColorMode): ColorMode {
  return mode === 'light' ? 'dark' : mode === 'dark' ? 'auto' : 'light'
}

export const useStyleStore = defineStore('style', () => {
  // Stored as a raw string (no JSON) so the inline script can read it without parsing.
  const colorMode = useStorage<ColorMode>(COLOR_MODE_STORAGE_KEY, 'auto', undefined, {
    serializer: {
      read: v => (COLOR_MODES.includes(v as ColorMode) ? v as ColorMode : 'auto'),
      write: v => v,
    },
  })
  const prefersDark = useMediaQuery('(prefers-color-scheme: dark)')
  const isDarkTheme = computed(() => resolveDark(colorMode.value, prefersDark.value))

  watchEffect(() => {
    const el = document.documentElement
    el.classList.toggle('dark', isDarkTheme.value)
    el.style.colorScheme = isDarkTheme.value ? 'dark' : 'light'
    el.dataset.colorMode = colorMode.value
    document.querySelector('meta[name="theme-color"]')
      ?.setAttribute('content', isDarkTheme.value ? THEME_COLOR.dark : THEME_COLOR.light)
  })

  const cycle = () => {
    colorMode.value = nextColorMode(colorMode.value)
  }

  const accent = useStorage<Accent>('nettools:accent', DEFAULT_ACCENT, undefined, {
    serializer: {
      read: v => (ACCENTS.includes(v as Accent) ? v as Accent : DEFAULT_ACCENT),
      write: v => v,
    },
  })

  // Favicon and boot-logo colour follow the accent (index.html applies them before the app loads).
  watchEffect(() => {
    const color = ACCENT_COLORS[accent.value][isDarkTheme.value ? 'dark' : 'light'].primary
    document.documentElement.style.setProperty('--nt-accent', color)
    document.querySelector('link[rel="icon"]')?.setAttribute('href', faviconDataUrl(color))
  })

  const isMenuCollapsed = useStorage('nettools:menu-collapsed', false)
  const isSmallScreen = useMediaQuery('(max-width: 700px)')

  return { colorMode, isDarkTheme, cycle, accent, isMenuCollapsed, isSmallScreen }
})
