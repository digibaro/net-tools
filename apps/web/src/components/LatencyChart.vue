<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { computed } from 'vue'

/** Dependency-free RTT chart: one point per sample; null = lost (drawn as a mark on the axis). */
const props = withDefaults(defineProps<{ samples: (number | null)[], height?: number, label: string }>(), { height: 120 })
const theme = useThemeVars()
const W = 600
const PAD = 4

const max = computed(() => Math.max(1, ...props.samples.filter((v): v is number => v !== null)) * 1.15)
const x = (i: number) => (props.samples.length <= 1 ? W / 2 : PAD + (i * (W - 2 * PAD)) / (props.samples.length - 1))
const y = (v: number) => props.height - PAD - (v / max.value) * (props.height - 2 * PAD)
const path = computed(() => {
  let d = ''
  let pen = false
  props.samples.forEach((v, i) => {
    if (v === null) {
      pen = false
      return
    }
    d += `${pen ? 'L' : 'M'}${x(i).toFixed(1)},${y(v).toFixed(1)} `
    pen = true
  })
  return d
})
const lost = computed(() => props.samples.map((v, i) => (v === null ? x(i) : null)).filter((v): v is number => v !== null))
</script>

<template>
  <svg :viewBox="`0 0 ${W} ${height}`" preserveAspectRatio="none" class="block h-auto w-full" role="img" :aria-label="label" :style="{ height: `${height}px` }">
    <line :x1="0" :x2="W" :y1="height - PAD" :y2="height - PAD" :stroke="theme.dividerColor" stroke-width="1" />
    <path :d="path" fill="none" :stroke="theme.primaryColor" stroke-width="2" vector-effect="non-scaling-stroke" />
    <circle v-for="(v, i) in samples" v-show="v !== null" :key="i" :cx="x(i)" :cy="v === null ? 0 : y(v)" r="2.5" :fill="theme.primaryColor" />
    <rect v-for="lx in lost" :key="`l${lx}`" :x="lx - 2" :y="height - PAD - 10" width="4" height="10" :fill="theme.errorColor" />
  </svg>
</template>
