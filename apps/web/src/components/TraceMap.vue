<script setup lang="ts">
import type { TraceHop } from '@/utils/trace'
import { NButton, useThemeVars } from 'naive-ui'
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiPost } from '@/api/client'
import { asnPath } from '@/utils/bgp'
import { landPath, MAP_VIEW_H, MAP_W, project } from './remote-map'
import { asPath, hopIps } from './trace-path'

/** AS path of a finished trace, and (when geolocation is available) the route on a world map. */
const props = defineProps<{ hops: TraceHop[], geo: boolean }>()
const { t } = useI18n()
const theme = useThemeVars()
const path = computed(() => asPath(props.hops))
interface GeoRow { ip: string, latitude: number | null, longitude: number | null, city: string, countryCode: string }
const located = shallowRef<{ ip: string, x: number, y: number, label: string }[]>([])
const land = ref('')
const loading = ref(false)
const failed = ref(false)
async function locate() {
  loading.value = true
  try {
    const [l, r] = await Promise.all([landPath(), apiPost<{ results: GeoRow[] }>('/v1/geo/batch', { ips: hopIps(props.hops) })])
    land.value = l
    const seen = new Set<string>()
    located.value = r.results.filter(g => g.latitude !== null && g.longitude !== null).flatMap((g) => {
      const [x, y] = project(g.longitude!, g.latitude!)
      const key = `${Math.round(x)},${Math.round(y)}`
      if (seen.has(key))
        return [] // consecutive hops in the same city collapse into one point
      seen.add(key)
      return [{ ip: g.ip, x, y, label: [g.ip, g.city, g.countryCode].filter(Boolean).join(' · ') }]
    })
  }
  catch {
    failed.value = true
  }
  finally {
    loading.value = false
  }
}
onMounted(() => {
  if (props.geo)
    void locate()
})
const line = computed(() => located.value.map(p => `${p.x},${p.y}`).join(' '))
</script>

<template>
  <div class="mt-3" data-testid="trace-path">
    <div v-if="path.length" class="mb-2 flex flex-wrap items-center gap-1 text-sm">
      <span class="mr-1 nt-muted">{{ t('trace.asPath') }}</span>
      <template v-for="(a, i) in path" :key="`${a}-${i}`">
        <span v-if="i" aria-hidden="true" class="nt-muted">→</span>
        <RouterLink :to="asnPath(a)" class="font-mono">
          AS{{ a }}
        </RouterLink>
      </template>
    </div>
    <svg v-if="located.length > 1" :viewBox="`0 0 ${MAP_W} ${MAP_VIEW_H}`" class="block h-auto w-full" role="img" :aria-label="t('trace.map', { n: located.length })" data-testid="trace-map">
      <path :d="land" :fill="theme.dividerColor" />
      <polyline :points="line" fill="none" :stroke="theme.primaryColor" stroke-width="2.5" stroke-linejoin="round" />
      <g v-for="(p, i) in located" :key="p.ip">
        <title>{{ i + 1 }}. {{ p.label }}</title>
        <circle :cx="p.x" :cy="p.y" r="7" :fill="i === located.length - 1 ? theme.successColor : theme.primaryColor" :stroke="theme.cardColor" stroke-width="2" />
      </g>
    </svg>
    <p v-else-if="geo && !loading && !failed && located.length <= 1" class="m-0 text-xs nt-muted">
      {{ t('trace.mapLocal') }}
    </p>
    <NButton v-if="failed" size="tiny" quaternary @click="failed = false; locate()">
      {{ t('trace.mapRetry') }}
    </NButton>
  </div>
</template>
