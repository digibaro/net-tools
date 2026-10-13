<script setup lang="ts">
import type { ColorMode } from '@/stores/style.store'
import { ChevronDown, DeviceDesktop, Moon, Sun } from '@vicons/tabler'
import { NButton, NButtonGroup, NDropdown, NIcon, NTooltip } from 'naive-ui'
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { COLOR_MODES, useStyleStore } from '@/stores/style.store'

const style = useStyleStore()
const { t } = useI18n()
const icons = { light: Sun, dark: Moon, auto: DeviceDesktop } as const

const label = computed(() => style.colorMode === 'auto'
  ? t('theme.autoCurrently', { current: t(style.isDarkTheme ? 'theme.dark' : 'theme.light') })
  : t('theme.label', { mode: t(`theme.${style.colorMode}`) }))

const options = computed(() => COLOR_MODES.map(mode => ({
  key: mode,
  label: t(`theme.${mode}`),
  icon: () => h(NIcon, { 'component': icons[mode], 'aria-hidden': 'true' }),
})))
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
    <NDropdown trigger="click" :options="options" :value="style.colorMode" @select="(key: ColorMode) => (style.colorMode = key)">
      <NButton quaternary size="tiny" class="px-0.5" :aria-label="t('theme.choose')" data-testid="theme-menu">
        <template #icon>
          <NIcon aria-hidden="true" :component="ChevronDown" />
        </template>
      </NButton>
    </NDropdown>
  </NButtonGroup>
</template>
