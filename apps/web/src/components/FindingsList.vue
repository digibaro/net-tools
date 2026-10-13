<script setup lang="ts">
import { NAlert } from 'naive-ui'
import { useI18n } from 'vue-i18n'

/** Problems (errors) and warnings as translated alerts; codes map to `email.findings.<code>`. */
defineProps<{ problems: string[], warnings: string[], okText: string }>()
const { t, te } = useI18n()
const text = (code: string) => (te(`email.findings.${code}`) ? t(`email.findings.${code}`) : code)
</script>

<template>
  <div class="mb-4 flex flex-col gap-2" data-testid="findings">
    <NAlert v-for="p in problems" :key="`p${p}`" type="error" :show-icon="true">
      {{ text(p) }}
    </NAlert>
    <NAlert v-for="w in warnings" :key="`w${w}`" type="warning">
      {{ text(w) }}
    </NAlert>
    <NAlert v-if="!problems.length && !warnings.length" type="success">
      {{ okText }}
    </NAlert>
  </div>
</template>
