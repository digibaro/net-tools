import type { components } from '@/api/schema'

export type DhcpResult = components['schemas']['DhcpResult']

/** Linux interface names: up to 15 chars, no slashes or spaces. Empty = default route. */
export const isInterfaceName = (s: string) => /^$|^[\w.:-]{1,15}$/.test(s.trim())

/** 86400 → "1 d", 3600 → "1 h", 90 → "90 s" */
export function formatLease(s: number): string {
  if (s <= 0)
    return '—'
  if (s % 86400 === 0)
    return `${s / 86400} d`
  if (s % 3600 === 0)
    return `${s / 3600} h`
  if (s % 60 === 0)
    return `${s / 60} min`
  return `${s} s`
}

export type Verdict = 'none' | 'single' | 'multiple'
export const verdict = (r: DhcpResult): Verdict => r.servers === 0 ? 'none' : r.servers === 1 ? 'single' : 'multiple'
