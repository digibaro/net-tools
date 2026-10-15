<script setup lang="ts">
import { NAlert } from 'naive-ui'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ResultActions from '@/components/ResultActions.vue'
import { useQueryParam } from '@/composables/useQueryParam'
import { useValidation } from '@/composables/useValidation'
import CCard from '@/ui/c-card.vue'
import CInputText from '@/ui/c-input-text.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'
import { analyzeIPv6 } from './ipv6-address-analyzer.service'

const { t } = useI18n()
const input = useQueryParam({ name: 'ip', defaultValue: 'fe80::21b:21ff:fe3a:4b5c' })
const validation = useValidation({ source: input, rules: [{ validator: (s: string) => !s.trim() || !!analyzeIPv6(s), message: t('tools.ipv6-address-analyzer.invalid') }] })
const r = computed(() => analyzeIPv6(input.value))
const k = (key: string) => t(`tools.ipv6-address-analyzer.fields.${key}`)
const yes = (b: boolean) => t(b ? 'common.yes' : 'common.no')
const rows = computed(() => {
  const a = r.value
  if (!a)
    return []
  const out = [
    { label: k('compressed'), value: a.compressed, mono: true },
    { label: k('expanded'), value: a.expanded, mono: true },
    { label: k('type'), value: `${a.rangeName} (${a.range}${a.rfc ? `, ${a.rfc}` : ''})`, noCopy: true },
    { label: k('scope'), value: t(`tools.ipv6-address-analyzer.scopes.${a.scope}`), noCopy: true },
    { label: k('reachable'), value: yes(a.globallyReachable), noCopy: true },
  ]
  if (a.flags)
    out.push({ label: k('flags'), value: [a.flags.transient ? 'T' : '', a.flags.prefix ? 'P' : '', a.flags.rp ? 'R' : ''].filter(Boolean).join(' ') || t('tools.ipv6-address-analyzer.permanent'), noCopy: true })
  if (a.iid) {
    out.push({ label: k('prefix64'), value: a.prefix64!, mono: true })
    out.push({ label: k('iid'), value: `${a.iid.value} — ${t(`tools.ipv6-address-analyzer.iid.${a.iid.kind}`)}`, noCopy: true })
    if (a.iid.mac)
      out.push({ label: k('mac'), value: `${a.iid.mac}${a.iid.macLocal ? ` (${t('tools.ipv6-address-analyzer.macLocal')})` : ''}`, mono: true })
    if (a.iid.ipv4)
      out.push({ label: k('isatap'), value: a.iid.ipv4, mono: true })
    out.push({ label: k('solicited'), value: a.solicitedNode!, mono: true })
  }
  for (const e of a.embedded)
    out.push({ label: t(`tools.ipv4-in-ipv6.kinds.${e.kind}`), value: e.detail ? `${e.ipv4} (${e.detail})` : e.ipv4, mono: true })
  out.push({ label: k('arpa'), value: a.arpa, mono: true })
  out.push({ label: k('binary'), value: a.binary, mono: true })
  return out
})
const asText = () => rows.value.map(x => `${x.label}: ${x.value}`).join('\n')
</script>

<template>
  <CCard>
    <CInputText v-model:value="input" :label="t('tools.ipv6-address-analyzer.input')" placeholder="2001:db8::1" :validation="validation" monospace test-id="input" />
  </CCard>
  <CCard v-if="r" :title="t('common.result')" data-testid="result">
    <NAlert v-if="r.iid?.kind === 'eui64'" type="warning" class="mb-3">
      {{ t('tools.ipv6-address-analyzer.eui64Warning') }}
    </NAlert>
    <CKeyValueList :rows="rows" />
    <div class="mt-3">
      <ResultActions :text="asText" :json="r" filename="ipv6-address.json" />
    </div>
  </CCard>
</template>
