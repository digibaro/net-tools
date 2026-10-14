import type { components } from '@/api/schema'

export type LatencyRow = components['schemas']['LatencyRow']

export const parseTargets = (s: string) => [...new Set(s.split(/[\s,]+/).map(x => x.trim()).filter(Boolean))]

/** Order rows by average RTT, unreachable last. */
export function ranked(rows: LatencyRow[]): LatencyRow[] {
  return [...rows].sort((a, b) => (a.stats.received ? 0 : 1) - (b.stats.received ? 0 : 1) || a.stats.avgMs - b.stats.avgMs)
}
