<script setup lang="ts">
import { NCheckbox, NFormItem, NInputNumber } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { calculateMtu, decodeLayers, ENCAPSULATIONS, encodeLayers } from './mtu-calculator.service'

const { t } = useI18n()
const mtu = useQueryParam({ name: 'mtu', defaultValue: 1500 })
const layersParam = useQueryParam({ name: 'layers', defaultValue: '' })
const layers = computed(() => decodeLayers(layersParam.value))
function setLayer(id: string, n: number) {
  layersParam.value = encodeLayers({ ...layers.value, [id]: n })
}
const result = computed(() => calculateMtu(mtu.value, layers.value))
const k = (key: string) => t(`tools.mtu-calculator.fields.${key}`)
const rows = computed(() => {
  const r = result.value
  if (!r)
    return []
  return [
    { label: k('overhead'), value: `${r.totalOverhead} bytes` },
    { label: k('innerMtu'), value: String(r.innerMtu) },
    { label: k('mssIpv4'), value: String(r.mssIpv4) },
    { label: k('mssIpv6'), value: String(r.mssIpv6) },
  ]
})
const asText = () => rows.value.map(r => `${r.label}: ${r.value}`).join('\n')
</script>

<template>
  <CCard>
    <NFormItem :label="t('tools.mtu-calculator.linkMtu')" :show-feedback="false" class="mb-3 max-w-48">
      <NInputNumber v-model:value="mtu" :min="68" :max="65535" :show-button="false" :input-props="{ 'aria-label': t('tools.mtu-calculator.linkMtu') }" data-testid="mtu" />
    </NFormItem>
    <fieldset class="m-0 border-0 p-0">
      <legend class="mb-2 font-semibold">
        {{ t('tools.mtu-calculator.encapsulations') }}
      </legend>
      <div class="grid gap-2 sm:grid-cols-2">
        <div v-for="e in ENCAPSULATIONS" :key="e.id" class="flex items-start gap-2">
          <NCheckbox
            :checked="(layers[e.id] ?? 0) > 0"
            :data-testid="`layer-${e.id}`"
            @update:checked="(c: boolean) => setLayer(e.id, c ? 1 : 0)"
          >
            <span>{{ t(`tools.mtu-calculator.layers.${e.id}`) }} <span class="nt-muted">(−{{ e.overhead }})</span></span>
            <div class="text-xs nt-muted">
              {{ e.breakdown }}
            </div>
          </NCheckbox>
          <NInputNumber
            v-if="e.stackable && (layers[e.id] ?? 0) > 0"
            :value="layers[e.id]" :min="1" :max="16" size="tiny" class="w-16" :show-button="false"
            :input-props="{ 'aria-label': t('tools.mtu-calculator.count', { layer: t(`tools.mtu-calculator.layers.${e.id}`) }) }"
            @update:value="(n: number | null) => setLayer(e.id, n ?? 1)"
          />
        </div>
      </div>
    </fieldset>
  </CCard>
  <CCard v-if="result" :title="t('common.result')" data-testid="result">
    <CKeyValueList :rows="rows" />
    <p v-if="result.belowIpv6Minimum" class="mb-0 text-sm" role="alert">
      {{ t('tools.mtu-calculator.ipv6Minimum') }}
    </p>
    <div class="mt-4">
      <ResultActions :text="asText" :json="result" filename="mtu.json" />
    </div>
  </CCard>
  <p v-else class="nt-muted">
    {{ t('tools.mtu-calculator.invalid') }}
  </p>
</template>
