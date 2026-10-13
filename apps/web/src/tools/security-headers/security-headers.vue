<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import { example } from './security-headers.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'q', defaultValue: '' })
const result = computed(() => example(input.value))
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.security-headers.input')" test-id="input" />
  </CCard>
  <CCard v-if="result" :title="t('common.result')">
    <span data-testid="result">{{ result }}</span>
  </CCard>
</template>
