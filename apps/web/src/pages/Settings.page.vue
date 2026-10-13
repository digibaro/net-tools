<script setup lang="ts">
import type { ColorMode } from '@/stores/style.store'
import { NButton, NFormItem, NRadioButton, NRadioGroup, NSelect } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { availableLocales, setLocale } from '@/plugins/i18n.plugin'
import { COLOR_MODES, useStyleStore } from '@/stores/style.store'
import { useToolStore } from '@/stores/tool.store'
import CCard from '@/ui/c-card.vue'

const { t, locale } = useI18n()
const style = useStyleStore()
const tools = useToolStore()
const localeOptions = computed(() => availableLocales.map(l => ({ label: t(`locales.${l}`), value: l })))
</script>

<template>
  <div class="mx-auto max-w-2xl">
    <h1 class="mb-5 mt-0 text-2xl font-semibold">
      {{ t('nav.settings') }}
    </h1>
    <CCard :title="t('settings.appearance')">
      <NFormItem :label="t('theme.title')" :show-feedback="false">
        <NRadioGroup :value="style.colorMode" name="color-mode" @update:value="(v: ColorMode) => (style.colorMode = v)">
          <NRadioButton v-for="mode in COLOR_MODES" :key="mode" :value="mode" :label="t(`theme.${mode}`)" :data-testid="`theme-radio-${mode}`" />
        </NRadioGroup>
      </NFormItem>
    </CCard>
    <CCard :title="t('settings.language')">
      <NSelect :value="locale" :options="localeOptions" :input-props="{ 'aria-label': t('settings.language') }" @update:value="setLocale" />
    </CCard>
    <CCard :title="t('home.favorites')">
      <p class="mt-0 nt-muted">
        {{ t('settings.favoritesCount', { count: tools.favorites.length }) }}
      </p>
      <NButton :disabled="!tools.favorites.length" @click="tools.favorites.forEach(f => tools.toggleFavorite(f))">
        {{ t('settings.clearFavorites') }}
      </NButton>
    </CCard>
  </div>
</template>
