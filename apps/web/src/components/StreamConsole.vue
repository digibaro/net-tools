<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { nextTick, ref, watch } from 'vue'

const props = withDefaults(defineProps<{ lines: string[], maxHeight?: string }>(), { maxHeight: '24rem' })
const theme = useThemeVars()
const el = ref<HTMLElement>()

watch(() => props.lines.length, async () => {
  await nextTick()
  if (el.value)
    el.value.scrollTop = el.value.scrollHeight
})
</script>

<template>
  <pre
    ref="el"
    class="m-0 overflow-auto rounded p-3 text-xs leading-5 font-mono"
    :style="{ maxHeight, background: theme.codeColor, color: theme.textColor2, border: `1px solid ${theme.dividerColor}` }"
    role="log"
    aria-live="polite"
  >{{ lines.join('\n') }}</pre>
</template>
