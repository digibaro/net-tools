<script setup lang="ts">
import { Copy } from '@vicons/tabler'
import { NButton, NIcon, NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useCopy } from '@/composables/useCopy'

export interface KeyValueRow {
  label: string
  value: string
  mono?: boolean
  /** Hide the per-row copy button. */
  noCopy?: boolean
}

defineProps<{ rows: KeyValueRow[] }>()
const { copy } = useCopy()
const { t } = useI18n()
</script>

<template>
  <dl class="nt-kv grid grid-cols-1 m-0 gap-x-4 gap-y-1 sm:grid-cols-[minmax(10rem,auto)_1fr]">
    <template v-for="row in rows" :key="row.label">
      <dt class="pt-1 text-sm nt-muted">
        {{ row.label }}
      </dt>
      <dd class="m-0 min-w-0 flex items-start gap-1 pb-1 sm:pb-0">
        <span class="min-w-0 break-all pt-1" :class="{ 'font-mono': row.mono }" :data-testid="`kv-${row.label}`">{{ row.value }}</span>
        <NTooltip v-if="!row.noCopy && row.value">
          <template #trigger>
            <NButton quaternary circle size="small" :aria-label="t('common.copyValue', { label: row.label })" @click="copy(row.value)">
              <template #icon>
                <NIcon aria-hidden="true" :component="Copy" />
              </template>
            </NButton>
          </template>
          {{ t('common.copy') }}
        </NTooltip>
      </dd>
    </template>
  </dl>
</template>
