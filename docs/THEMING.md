# Theming — Light / Dark / Auto

it-tools toggles a boolean dark flag. NetTools needs a **three-state** preference:

| Preference | Behaviour |
|---|---|
| `light` | Always light |
| `dark` | Always dark |
| `auto` (default) | Follows `prefers-color-scheme`, **live-updates** when the OS switches |

## 1. Store

```ts
// apps/web/src/stores/style.store.ts
import { defineStore } from 'pinia'
import { useMediaQuery, useStorage } from '@vueuse/core'

export type ColorMode = 'light' | 'dark' | 'auto'
const STORAGE_KEY = 'nettools:color-mode'

export const useStyleStore = defineStore('style', () => {
  const colorMode = useStorage<ColorMode>(STORAGE_KEY, 'auto')
  const prefersDark = useMediaQuery('(prefers-color-scheme: dark)')
  const isDarkTheme = computed(() =>
    colorMode.value === 'auto' ? prefersDark.value : colorMode.value === 'dark')

  // keep <html> in sync for UnoCSS `dark:` variants and native controls
  watchEffect(() => {
    const el = document.documentElement
    el.classList.toggle('dark', isDarkTheme.value)
    el.style.colorScheme = isDarkTheme.value ? 'dark' : 'light'
    el.dataset.colorMode = colorMode.value
    document.querySelector('meta[name="theme-color"]')
      ?.setAttribute('content', isDarkTheme.value ? '#18181c' : '#ffffff')
  })

  const cycle = () => {
    colorMode.value = colorMode.value === 'light' ? 'dark' : colorMode.value === 'dark' ? 'auto' : 'light'
  }

  // existing it-tools style concerns
  const isMenuCollapsed = useStorage('nettools:menu-collapsed', false)
  const isSmallScreen = useMediaQuery('(max-width: 700px)')

  return { colorMode, isDarkTheme, cycle, isMenuCollapsed, isSmallScreen }
})
```

## 2. Naive UI wiring

```vue
<!-- App.vue -->
<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import { darkThemeOverrides, lightThemeOverrides } from './themes'
const styleStore = useStyleStore()
const theme = computed(() => (styleStore.isDarkTheme ? darkTheme : null))
const overrides = computed(() => (styleStore.isDarkTheme ? darkThemeOverrides : lightThemeOverrides))
</script>

<template>
  <n-config-provider :theme="theme" :theme-overrides="overrides">
    <n-global-style />
    <n-message-provider><n-notification-provider>
      <component :is="layout"><router-view /></component>
    </n-notification-provider></n-message-provider>
  </n-config-provider>
</template>
```

`themes.ts` builds the overrides with `themeOverrides(accent, dark)`.

### Accent colours

Users pick an accent in Settings → Appearance or the header theme menu: green (default), blue,
indigo, purple, orange or red. The choice is stored per browser (`nettools:accent`) in the style
store; `App.vue` passes `themeOverrides(style.accent, style.isDarkTheme)` to `NConfigProvider`.

Each preset in `ACCENT_COLORS` has two variants with primary / hover / pressed / suppl shades:
a dark shade for light mode and a light tint for dark mode. `themes.test.ts` enforces
≥ 4.5:1 contrast for every preset:

- light shades (primary and pressed) against white — links, and white text on primary buttons;
- dark tints against Naive's dark body, card and popover surfaces, and against the dark text
  Naive puts on primary buttons.

`e2e/appearance.e2e.spec.ts` runs axe on Settings and a tool page for every accent in both
themes. To add a preset, add it to `ACCENTS` / `ACCENT_COLORS`, a label under `theme.accents` in
`locales/en.yml`, and run both tests.

## 3. UnoCSS

```ts
// uno.config.ts
export default defineConfig({
  presets: [presetUno({ dark: 'class' }), presetIcons(), presetScrollbar()],
  theme: { colors: { primary: 'var(--nt-primary)' } },
  shortcuts: {
    'nt-surface': 'bg-white dark:bg-[#232326]',
    'nt-muted': 'text-neutral-500 dark:text-neutral-400',
  },
})
```

Prefer Naive UI CSS vars (`useThemeVars()`) inside components; use `dark:` utilities only for layout
chrome. **No hard-coded hex colors in tool components.** Status colors (success / warning / error /
info) come from Naive's theme vars so they adapt.

## 4. No flash of wrong theme

Inline script in `index.html` `<head>`, before CSS loads (its sha256 goes in the CSP):

```html
<meta name="theme-color" content="#ffffff">
<script>
  (function () {
    try {
      var m = localStorage.getItem('nettools:color-mode') || '"auto"'
      m = JSON.parse(m)
      var d = m === 'dark' || (m === 'auto' && matchMedia('(prefers-color-scheme: dark)').matches)
      var e = document.documentElement
      if (d) e.classList.add('dark')
      e.style.colorScheme = d ? 'dark' : 'light'
    } catch (_) {}
  })()
</script>
```

(`useStorage` serializes strings as JSON — keep the parse in sync, or configure `useStorage` with a
raw string serializer and drop `JSON.parse`.)

## 5. Switcher UI

- Header button (`ThemeSwitcher.vue`) shows icon for the **preference**: sun (light), moon (dark),
  monitor/half-circle (auto). Click cycles; long-press / right-click or a dropdown chevron opens a
  3-item menu. Tooltip: "Theme: Auto (currently dark)".
- Settings page: radio group Light / Dark / Auto.
- Keyboard: command palette entries "Theme: Light/Dark/Auto".
- `aria-label` reflects state.

## 6. Charts, code, terminal output

- ECharts: register `light` and `dark` themes; re-render on `isDarkTheme` change.
- `StreamConsole.vue` (traceroute/ping output) uses `--n-code-color`/`--n-color-embedded` theme vars.
- Highlight.js: load both stylesheets scoped under `html.dark` / `html:not(.dark)`.
- Map tiles (Globalping map): use tiles with dark variant, or invert via CSS filter on dark.

## 7. Tests

Playwright (`theme.e2e.spec.ts`):
1. `emulateMedia({ colorScheme: 'dark' })` + no storage → `html.dark` present.
2. Switch OS scheme at runtime while in auto → class flips without reload.
3. Choose `light` while OS dark → stays light after reload.
4. No flash: screenshot of first paint background matches theme (check `html` class at
   `DOMContentLoaded`).
5. axe-core contrast check on Home in both themes.
