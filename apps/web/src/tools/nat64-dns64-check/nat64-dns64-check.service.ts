import type { components } from '@/api/schema'

export type Dns64Result = components['schemas']['Dns64Result']

/** Resolver field: empty = the server's system resolver; otherwise an IP address (port optional). */
export const isResolver = (s: string) => !s.trim() || /^(?:[\d.]+|\[?[\da-f:]+\]?)(?::\d{1,5})?$/i.test(s.trim())

export type Verdict = 'wellKnown' | 'networkSpecific' | 'none'
export function verdict(r: Dns64Result): Verdict {
  if (!r.dns64)
    return 'none'
  return r.prefixes.every(p => p.wellKnown) ? 'wellKnown' : 'networkSpecific'
}
