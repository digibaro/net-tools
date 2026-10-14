import type { components } from '@/api/schema'

export type TlsExpiry = components['schemas']['TlsExpiry']

/** One host[:port] per line (or comma separated), deduplicated. */
export function parseTargets(text: string): string[] {
  return [...new Set(text.split(/[\n,]+/).map(s => s.trim().replace(/^https?:\/\//, '').replace(/\/.*$/, '')).filter(Boolean))]
}

/** Soonest expiry first; errors last. */
export function sortRows(rows: TlsExpiry['rows']): TlsExpiry['rows'] {
  return [...rows].sort((a, b) => (a.daysLeft ?? Infinity) - (b.daysLeft ?? Infinity))
}
