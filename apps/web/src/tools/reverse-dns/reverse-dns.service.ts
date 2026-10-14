import type { components } from '@/api/schema'
import { parseIP } from '@/utils/ip'

export type DnsReverse = components['schemas']['DnsReverse']

export const isIpAddress = (s: string) => parseIP(s) !== undefined

/** One-line verdict: every PTR name forward-confirms, some do, none do, or no PTR. */
export function verdict(r: Pick<DnsReverse, 'names'>): 'confirmed' | 'partial' | 'unconfirmed' | 'none' {
  if (!r.names.length)
    return 'none'
  const ok = r.names.filter(n => n.forwardConfirmed).length
  return ok === r.names.length ? 'confirmed' : ok ? 'partial' : 'unconfirmed'
}
