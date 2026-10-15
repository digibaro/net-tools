import type { components } from '@/api/schema'

export type CacheResult = components['schemas']['CacheResult']

/** 3600 → "1 h", 90 → "1 min 30 s", -1 → undefined (not cacheable / unknown). */
export function duration(s: number): string | undefined {
  if (s < 0)
    return undefined
  if (s === 0)
    return '0 s'
  const parts: string[] = []
  for (const [unit, n] of [['d', 86400], ['h', 3600], ['min', 60], ['s', 1]] as const) {
    if (s >= n) {
      parts.push(`${Math.floor(s / n)} ${unit}`)
      s %= n
    }
    if (parts.length === 2)
      break
  }
  return parts.join(' ')
}
