import type { components } from '@/api/schema'

export type IperfInterval = components['schemas']['IperfInterval']
export type IperfSummary = components['schemas']['IperfSummary']

/** 940.2 → "940 Mbit/s", 12409.5 → "12.4 Gbit/s", 0.35 → "350 kbit/s" */
export function formatRate(mbps: number): string {
  if (mbps >= 1000)
    return `${(mbps / 1000).toFixed(mbps >= 10000 ? 1 : 2)} Gbit/s`
  if (mbps >= 1)
    return `${mbps >= 100 ? Math.round(mbps) : mbps.toFixed(1)} Mbit/s`
  return `${Math.round(mbps * 1000)} kbit/s`
}

/** 3109027840 → "2.9 GiB" */
export function formatBytes(n: number): string {
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${i === 0 ? v : v.toFixed(1)} ${units[i]}`
}
