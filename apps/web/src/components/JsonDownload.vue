<script setup lang="ts">
import { Download } from '@vicons/tabler'
import { NButton, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { downloadText, toJson } from '@/utils/download'

const props = defineProps<{ data: unknown, filename: string }>()
const { t } = useI18n()
</script>

<template>
  <NButton secondary size="small" @click="downloadText(props.filename, toJson(props.data))">
    <template #icon>
      <NIcon aria-hidden="true" :component="Download" />
    </template>
    {{ t('common.downloadJson') }}
  </NButton>
</template>
