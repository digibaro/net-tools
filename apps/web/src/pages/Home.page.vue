<script setup lang="ts">
import type { VersionInfo } from '@/api/client'
import { NCard, NText } from 'naive-ui'
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getVersion } from '@/api/client'

const { t } = useI18n()
const version = ref<VersionInfo>()
const error = ref<string>()

onMounted(async () => {
  try {
    version.value = await getVersion()
  }
  catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  }
})
</script>

<template>
  <main class="mx-auto max-w-2xl px-4 py-12">
    <NCard>
      <h1 class="m-0 text-2xl font-semibold">
        {{ t('app.name') }}
      </h1>
      <p class="mb-6 mt-2">
        <NText depth="3">
          {{ t('app.tagline') }}
        </NText>
      </p>
      <p>{{ t('home.placeholder') }}</p>
      <p data-testid="api-status">
        <NText v-if="version" depth="3">
          {{ t('home.apiVersion', { version: version.version, commit: version.commit }) }}
        </NText>
        <NText v-else-if="error" type="error">
          {{ t('home.apiError', { message: error }) }}
        </NText>
        <NText v-else depth="3">
          {{ t('home.apiLoading') }}
        </NText>
      </p>
    </NCard>
  </main>
</template>
