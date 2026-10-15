<script setup lang="ts">
import { NButton, NButtonGroup } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { setLocale } from '@/plugins/i18n.plugin'

// Short labels in the top bar; the full language name is the accessible label.
const LANGUAGES = [
  { code: 'nl', label: 'NL' },
  { code: 'en', label: 'UK' },
] as const

const { t, locale } = useI18n()
</script>

<template>
  <NButtonGroup size="small" role="group" :aria-label="t('settings.language')" data-testid="locale-switch">
    <NButton
      v-for="l in LANGUAGES"
      :key="l.code"
      :type="locale === l.code ? 'primary' : 'default'"
      :secondary="locale !== l.code"
      :aria-pressed="locale === l.code"
      :aria-label="t(`locales.${l.code}`)"
      :title="t(`locales.${l.code}`)"
      class="min-w-9 px-2 font-semibold"
      :data-testid="`locale-${l.code}`"
      @click="setLocale(l.code)"
    >
      {{ l.label }}
    </NButton>
  </NButtonGroup>
</template>
