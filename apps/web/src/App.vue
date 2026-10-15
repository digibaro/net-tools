<script setup lang="ts">
import { darkTheme, NConfigProvider, NGlobalStyle, NMessageProvider } from 'naive-ui'
import { computed, onMounted, watchEffect } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import BaseLayout from './layouts/base.layout.vue'
import ToolLayout from './layouts/tool.layout.vue'
import { useCapabilitiesStore } from './stores/capabilities.store'
import { useStyleStore } from './stores/style.store'
import { themeOverrides } from './themes'

const style = useStyleStore()
const caps = useCapabilitiesStore()
const route = useRoute()
const { t } = useI18n()

const theme = computed(() => (style.isDarkTheme ? darkTheme : null))
const overrides = computed(() => themeOverrides(style.accent, style.isDarkTheme))

watchEffect(() => {
  const name = route.meta.tool?.name ?? (route.meta.titleKey ? t(route.meta.titleKey) : undefined)
  document.title = name ? `${name} · ${t('app.name')}` : `${t('app.name')} · ${t('app.tagline')}`
})

onMounted(() => {
  void caps.load()
})
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="overrides">
    <NGlobalStyle />
    <NMessageProvider>
      <BaseLayout>
        <RouterView v-slot="{ Component, route: r }">
          <ToolLayout v-if="r.meta.tool" :key="r.path" :tool="r.meta.tool">
            <component :is="Component" />
          </ToolLayout>
          <component :is="Component" v-else />
        </RouterView>
      </BaseLayout>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style>
/* Radio button groups (segmented choices) wrap instead of running off narrow screens. */
.n-radio-group.n-radio-group {
  display: flex;
  flex-wrap: wrap;
  row-gap: 4px;
  height: auto;
  max-width: 100%;
}

/* When they can wrap, every choice is its own outlined button (the shared dividers would end up
   at the end of a row and leave the next row without a left border). */
@media (max-width: 640px) {
  .n-radio-group.n-radio-group {
    gap: 6px;
  }
  .n-radio-group .n-radio-group__splitor {
    display: none;
  }
  .n-radio-group .n-radio-button.n-radio-button {
    border: 1px solid var(--n-button-border-color);
    border-radius: var(--n-button-border-radius);
  }
  .n-radio-group .n-radio-button.n-radio-button--checked {
    border-color: var(--n-button-border-color-active);
  }
}

/* Print / "save as PDF": only the tool and its results, on a light page. */
@media print {
  .nt-no-print,
  [data-testid='sidebar'],
  .n-layout-header,
  .n-button:not(.n-tag) { display: none !important; }
  html, body, .n-layout, .n-layout-scroll-container { background: #fff !important; color: #000 !important; height: auto !important; overflow: visible !important; }
  .n-layout, .n-layout-content { position: static !important; }
  [role='region'] { max-height: none !important; overflow: visible !important; }
  a[href^='http']::after { content: ' (' attr(href) ')'; font-size: 0.8em; }
}
</style>
