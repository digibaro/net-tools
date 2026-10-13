<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { toJson } from '@/utils/download'
import CopyButton from './CopyButton.vue'
import JsonDownload from './JsonDownload.vue'

/** FR-13: copy as text, copy as JSON, download JSON. */
const props = defineProps<{ text: string | (() => string), json: unknown, filename: string }>()
const { t } = useI18n()
</script>

<template>
  <div class="flex flex-wrap gap-2">
    <CopyButton :text="props.text" :label="t('common.copyText')" />
    <CopyButton :text="() => toJson(props.json)" :label="t('common.copyJson')" />
    <JsonDownload :data="props.json" :filename="props.filename" />
  </div>
</template>
