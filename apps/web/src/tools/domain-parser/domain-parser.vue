<script setup lang="ts">
import type { DomainParts } from './domain-parser.service'
import { NAlert } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { analyzeDomain } from './domain-parser.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'q', defaultValue: 'https://blog.user.github.io/post' })
const validation = useValidation({ source: input, rules: [{ validator: (v: string) => !!analyzeDomain(v), message: t('tools.domain-parser.invalid') }] })
const result = computed(() => analyzeDomain(input.value))
const differs = computed(() => result.value && result.value.withPrivate.domain !== result.value.icannOnly.domain)

function rowsFor(p: DomainParts) {
  const k = (key: string) => t(`tools.domain-parser.fields.${key}`)
  const v = (s: string) => s || '—'
  return [
    { label: k('hostname'), value: p.hostname, mono: true },
    { label: k('subdomain'), value: v(p.subdomain), mono: true, noCopy: !p.subdomain },
    { label: k('domain'), value: v(p.domain), mono: true, noCopy: !p.domain },
    { label: k('domainWithoutSuffix'), value: v(p.domainWithoutSuffix), mono: true, noCopy: !p.domainWithoutSuffix },
    { label: k('publicSuffix'), value: v(p.publicSuffix), mono: true, noCopy: !p.publicSuffix },
    { label: k('suffixType'), value: t(`tools.domain-parser.suffixTypes.${p.suffixType}`), noCopy: true },
  ]
}
const asText = () => result.value ? rowsFor(result.value.withPrivate).map(r => `${r.label}: ${r.value}`).join('\n') : ''
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.domain-parser.input')" placeholder="www.example.co.uk" :validation="validation" monospace test-id="input" />
  </CCard>
  <NAlert v-if="result?.isIp" type="info" class="mb-4">
    {{ t('tools.domain-parser.isIp') }}
  </NAlert>
  <template v-else-if="result">
    <CCard :title="t('tools.domain-parser.withPrivate')" data-testid="result">
      <CKeyValueList :rows="rowsFor(result.withPrivate)" />
      <div class="mt-4">
        <ResultActions :text="asText" :json="result" filename="domain.json" />
      </div>
    </CCard>
    <CCard v-if="differs" :title="t('tools.domain-parser.icannOnly')" data-testid="icann">
      <p class="mt-0 text-sm nt-muted">
        {{ t('tools.domain-parser.differs') }}
      </p>
      <CKeyValueList :rows="rowsFor(result.icannOnly)" />
    </CCard>
  </template>
</template>
