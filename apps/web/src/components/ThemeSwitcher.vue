<script setup lang="ts">
import type { ColorMode } from '@/stores/style.store'
import type { Accent } from '@/themes'
import { ChevronDown, DeviceDesktop, Moon, Sun } from '@vicons/tabler'
import { NButton, NButtonGroup, NDropdown, NIcon, NTooltip } from 'naive-ui'
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { COLOR_MODES, useStyleStore } from '@/stores/style.store'
import { ACCENTS } from '@/themes'
import AccentSwatch from './AccentSwatch.vue'

const style = useStyleStore()
const { t } = useI18n()
const icons = { light: Sun, dark: Moon, auto: DeviceDesktop } as const

const label = computed(() => style.colorMode === 'auto'
  ? t('theme.autoCurrently', { current: t(style.isDarkTheme ? 'theme.dark' : 'theme.light') })
  : t('theme.label', { mode: t(`theme.${style.colorMode}`) }))

const options = computed(() => [
  ...COLOR_MODES.map(mode => ({
    key: mode,
    label: t(`theme.${mode}`),
    icon: () => h(NIcon, { 'component': icons[mode], 'aria-hidden': 'true' }),
  })),
  { type: 'divider', key: 'divider' },
  {
    key: 'accent',
    label: t('theme.accent'),
    icon: () => h(AccentSwatch, { accent: style.accent }),
    children: ACCENTS.map(a => ({
      key: `accent:${a}`,
      label: t(`theme.accents.${a}`),
      icon: () => h(AccentSwatch, { accent: a }),
    })),
  },
])

function onSelect(key: string) {
  if (key.startsWith('accent:'))
    style.accent = key.slice('accent:'.length) as Accent
  else
    style.colorMode = key as ColorMode
}
</script>

<template>
  <NButtonGroup>
    <NTooltip>
      <template #trigger>
        <NButton quaternary circle :aria-label="label" data-testid="theme-toggle" @click="style.cycle()">
          <template #icon>
            <NIcon aria-hidden="true" :component="icons[style.colorMode]" />
          </template>
        </NButton>
      </template>
      {{ label }}
    </NTooltip>
    <NDropdown trigger="click" :options="options" :value="style.colorMode" @select="onSelect">
      <NButton quaternary size="small" class="min-h-6 min-w-6 px-1" :aria-label="t('theme.choose')" data-testid="theme-menu">
        <template #icon>
          <NIcon aria-hidden="true" :component="ChevronDown" />
        </template>
      </NButton>
    </NDropdown>
  </NButtonGroup>
</template>
