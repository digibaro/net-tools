<script setup lang="ts">
import { Refresh } from '@vicons/tabler'
import { NButton, NIcon } from 'naive-ui'
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { macToEui64, parseMac } from '@/utils/mac'
import { computeGlobalId, ntpTimestamp, parseGlobalId, randomEui64, ulaFromGlobalId } from './ipv6-ula-generator.service'

const { t } = useI18n()
const gid = useQueryParam({ name: 'gid', defaultValue: '' })
const mac = useQueryParam({ name: 'mac', defaultValue: '' })

const macValidation = computed(() => (mac.value && parseMac(mac.value) === undefined)
  ? { isValid: false, status: 'error' as const, message: t('tools.ipv6-ula-generator.invalidMac') }
  : undefined)

async function generate() {
  const m = parseMac(mac.value)
  const eui = m === undefined ? randomEui64() : macToEui64(m)
  gid.value = (await computeGlobalId(ntpTimestamp(Date.now()), eui)).toString(16).padStart(10, '0')
}

onMounted(() => {
  if (parseGlobalId(gid.value) === undefined)
    void generate()
})

const result = computed(() => {
  const g = parseGlobalId(gid.value)
  return g === undefined ? undefined : ulaFromGlobalId(g)
})
const rows = computed(() => {
  const r = result.value
  if (!r)
    return []
  const k = (key: string) => t(`tools.ipv6-ula-generator.fields.${key}`)
  return [
    { label: k('prefix'), value: r.prefix, mono: true },
    { label: k('globalId'), value: r.globalId, mono: true },
    { label: k('subnets'), value: `${r.firstSubnet} – ${r.lastSubnet}`, mono: true },
    { label: k('subnetCount'), value: r.subnetCount, noCopy: true },
    { label: k('examples'), value: r.exampleSubnets.join(', '), mono: true },
  ]
})
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText v-model:value="mac" :label="t('tools.ipv6-ula-generator.mac')" placeholder="00:1a:2b:3c:4d:5e" :validation="macValidation" monospace test-id="mac" />
    <p class="mt-0 text-sm nt-muted">
      {{ t('tools.ipv6-ula-generator.hint') }}
    </p>
    <NButton type="primary" :disabled="!!macValidation" data-testid="generate" @click="generate">
      <template #icon>
        <NIcon aria-hidden="true" :component="Refresh" />
      </template>
      {{ t('common.generate') }}
    </NButton>
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="ula.json" />
    </div>
  </CCard>
</template>
