<script setup lang="ts">
import { Copy } from '@vicons/tabler'
import { NButton, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useCopy } from '@/composables/useCopy'

const props = defineProps<{ text: string | (() => string), label?: string, size?: 'tiny' | 'small' | 'medium' }>()
const { copy } = useCopy()
const { t } = useI18n()

function onClick() {
  void copy(typeof props.text === 'function' ? props.text() : props.text)
}
</script>

<template>
  <NButton secondary :size="size ?? 'small'" @click="onClick">
    <template #icon>
      <NIcon aria-hidden="true" :component="Copy" />
    </template>
    {{ label ?? t('common.copy') }}
  </NButton>
</template>
