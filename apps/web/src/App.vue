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
