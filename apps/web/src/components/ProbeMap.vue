<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { hasPosition, landPath, MAP_VIEW_H, MAP_W, project } from './remote-map'

/** Probe locations on a world map, coloured by outcome; a dot jumps to its result. */
export interface MapPoint { id: string, latitude: number, longitude: number, label: string, verdict: 'up' | 'down' | 'error' }
const props = defineProps<{ points: MapPoint[] }>()
const emit = defineEmits<{ select: [id: string] }>()
const { t } = useI18n()
const theme = useThemeVars()
const land = ref('')
onMounted(async () => (land.value = await landPath()))
const color = computed(() => ({ up: theme.value.successColor, down: theme.value.errorColor, error: theme.value.warningColor }))
const dots = computed(() => props.points.filter(hasPosition).map((p) => {
  const [x, y] = project(p.longitude, p.latitude)
  return { ...p, x, y }
}))
</script>

<template>
  <svg :viewBox="`0 0 ${MAP_W} ${MAP_VIEW_H}`" class="mb-3 block h-auto w-full" role="img" :aria-label="t('remote.map', { n: dots.length })" data-testid="probe-map">
    <path :d="land" :fill="theme.dividerColor" />
    <g v-for="d in dots" :key="d.id" class="cursor-pointer" @click="emit('select', d.id)">
      <title>{{ d.label }}</title>
      <circle :cx="d.x" :cy="d.y" r="9" :fill="color[d.verdict]" :stroke="theme.cardColor" stroke-width="2" fill-opacity="0.85" />
    </g>
  </svg>
</template>
