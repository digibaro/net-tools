import type { components } from '@/api/schema'

export type DnsPropagation = components['schemas']['DnsPropagation']
export const PROPAGATION_TYPES = ['A', 'AAAA', 'CNAME', 'MX', 'NS', 'TXT', 'SOA', 'CAA', 'SRV', 'PTR'] as const

/** Group label: A, B, C… (majority first). */
export const groupLabel = (i: number) => (i < 0 ? '—' : String.fromCharCode(65 + (i % 26)))

/** Share of answering resolvers that agree with the majority, 0–100. */
export function agreement(r: Pick<DnsPropagation, 'groups'>): number {
  const total = r.groups.reduce((n, g) => n + g.count, 0)
  return total ? Math.round(((r.groups[0]?.count ?? 0) / total) * 100) : 0
}
