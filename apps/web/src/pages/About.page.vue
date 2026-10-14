<script setup lang="ts">
import type { VersionInfo } from '@/api/client'
import { NA, NTag } from 'naive-ui'
import { computed, onMounted, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getVersion } from '@/api/client'
import { useCapabilitiesStore } from '@/stores/capabilities.store'
import CCard from '@/ui/c-card.vue'
import CKeyValueList from '@/ui/c-key-value-list.vue'

const { t } = useI18n()
const caps = useCapabilitiesStore()
const apiVersion = shallowRef<VersionInfo>()
const apiError = shallowRef(false)

onMounted(async () => {
  void caps.load()
  try {
    apiVersion.value = await getVersion()
  }
  catch {
    apiError.value = true
  }
})

const rows = computed(() => [
  { label: t('about.webVersion'), value: __APP_VERSION__, mono: true },
  { label: t('about.apiVersion'), value: apiVersion.value ? `${apiVersion.value.version} (${apiVersion.value.commit})` : apiError.value ? t('about.apiUnreachable') : '…', mono: true },
  { label: t('about.buildDate'), value: apiVersion.value?.buildDate ?? '—', mono: true },
  { label: t('about.profile'), value: caps.capabilities ? caps.capabilities.mode.profile : '—', noCopy: true },
  { label: t('about.privateTargets'), value: caps.capabilities ? t(caps.capabilities.mode.allowPrivateTargets ? 'common.allowed' : 'common.denied') : '—', noCopy: true },
  { label: t('about.lanMode'), value: caps.capabilities ? t(caps.capabilities.mode.lan ? 'common.on' : 'common.off') : '—', noCopy: true },
])

const capabilityList = computed(() => Object.entries(caps.capabilities?.tools ?? {}).sort(([a], [b]) => a.localeCompare(b)))

const attributions = [
  { name: 'it-tools', url: 'https://github.com/CorentinTh/it-tools', license: 'GPL-3.0' },
  { name: 'Vue', url: 'https://vuejs.org', license: 'MIT' },
  { name: 'Naive UI', url: 'https://www.naiveui.com', license: 'MIT' },
  { name: 'Tabler Icons', url: 'https://tabler.io/icons', license: 'MIT' },
  { name: 'Fuse.js', url: 'https://www.fusejs.io', license: 'Apache-2.0' },
  { name: 'ipaddr.js', url: 'https://github.com/whitequark/ipaddr.js', license: 'MIT' },
  { name: 'ip-address', url: 'https://github.com/beaugunderson/ip-address', license: 'MIT' },
  { name: 'tldts', url: 'https://github.com/remusao/tldts', license: 'MIT' },
  { name: 'UAParser.js', url: 'https://github.com/faisalman/ua-parser-js', license: 'MIT (v1)' },
  { name: '@peculiar/x509', url: 'https://github.com/PeculiarVentures/x509', license: 'MIT' },
  { name: 'node-qrcode', url: 'https://github.com/soldair/node-qrcode', license: 'MIT' },
  { name: 'oui-data (IEEE OUI registry)', url: 'https://github.com/silverwind/oui-data', license: 'BSD-2-Clause' },
  { name: 'IANA registries', url: 'https://www.iana.org/protocols', license: 'IANA terms' },
  { name: 'Go, chi', url: 'https://github.com/go-chi/chi', license: 'BSD-3-Clause / MIT' },
]
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <h1 class="mb-2 mt-0 text-2xl font-semibold">
      {{ t('nav.about') }}
    </h1>
    <p class="mt-0 nt-muted">
      {{ t('about.intro') }}
    </p>
    <CCard :title="t('about.build')">
      <CKeyValueList :rows="rows" />
    </CCard>
    <CCard :title="t('about.capabilities')">
      <p v-if="!capabilityList.length" class="m-0 nt-muted">
        {{ caps.state === 'unreachable' ? t('about.apiUnreachable') : t('about.noServerTools') }}
      </p>
      <div v-else class="flex flex-wrap gap-2">
        <NTag v-for="[id, cap] in capabilityList" :key="id" :type="cap.enabled ? 'success' : 'default'" size="small">
          {{ id }}<template v-if="!cap.enabled">
            — {{ t(`capabilities.reasons.${cap.reason ?? 'unavailable'}`) }}
          </template>
        </NTag>
      </div>
    </CCard>
    <CCard :title="t('about.attributions')">
      <p class="mt-0">
        {{ t('about.license') }}
      </p>
      <ul class="m-0 pl-5">
        <li v-for="a in attributions" :key="a.name">
          <NA :href="a.url" target="_blank" rel="noopener noreferrer">
            {{ a.name }}
          </NA> <span class="nt-muted">— {{ a.license }}</span>
        </li>
      </ul>
    </CCard>
  </div>
</template>
