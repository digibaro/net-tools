<script setup lang="ts">
import { NButton, NResult } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const { t } = useI18n()
const router = useRouter()
</script>

<template>
  <NResult status="404" :title="t('notFound.title')" :description="t('notFound.description')" class="mt-12" data-testid="not-found">
    <template #footer>
      <NButton type="primary" @click="router.push('/')">
        {{ t('notFound.back') }}
      </NButton>
    </template>
  </NResult>
</template>
