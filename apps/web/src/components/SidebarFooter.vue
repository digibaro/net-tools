<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { useI18n } from 'vue-i18n'

// Copyright holder and licence of this distribution (see LICENSE).
const COPYRIGHT = { year: 2026, holder: 'digibaro' }
const LICENSE = { name: 'GPL-3.0-or-later', url: 'https://github.com/digibaro/net-tools-docker/blob/main/LICENSE' }

const { t } = useI18n()
const theme = useThemeVars()
const version = __APP_VERSION__
</script>

<template>
  <footer class="mt-auto px-4 pb-4 pt-6 text-xs leading-relaxed" :style="{ color: theme.textColor3 }" data-testid="sidebar-footer">
    <div>{{ t('app.name') }} {{ t('footer.version', { version }) }}</div>
    <div>{{ t('footer.copyright', { year: COPYRIGHT.year, holder: COPYRIGHT.holder }) }}</div>
    <div>
      {{ t('footer.license') }}
      <a :href="LICENSE.url" target="_blank" rel="noopener noreferrer" :style="{ color: 'inherit' }" class="underline">{{ LICENSE.name }}</a>
    </div>
  </footer>
</template>
