import type { components } from '@/api/schema'

export type DnsAxfr = components['schemas']['DnsAxfr']
export type AxfrServer = DnsAxfr['servers'][number]

export function serverState(s: AxfrServer): 'open' | 'refused' | 'error' {
  if (s.allowed)
    return 'open'
  return s.error.startsWith('refused') || s.error === '' ? 'refused' : 'error'
}

/** "A 12, MX 2, NS 4" sorted by count. */
export const typeSummary = (types: Record<string, number>) => Object.entries(types).sort((a, b) => b[1] - a[1]).map(([t, n]) => `${t} ${n}`).join(', ')
