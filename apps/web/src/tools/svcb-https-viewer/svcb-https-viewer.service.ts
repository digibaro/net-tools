import type { components } from '@/api/schema'

export type SvcbResult = components['schemas']['SvcbResult']
export type SvcbRecord = components['schemas']['SvcbRecord']

/** Domain names, including underscore labels such as _dns.resolver.arpa or _443._https.example.com. */
export const isOwnerName = (s: string) => /^(?:[\w-]{1,63}\.)*[\w-]{1,63}\.?$/.test(s.trim()) && s.trim().length <= 253

/** Short summary of what a service record offers (for the record heading). */
export function offers(r: SvcbRecord): string[] {
  const out: string[] = []
  const alpn = r.params.find(p => p.key === 'alpn')?.value
  if (alpn)
    out.push(...alpn.split(','))
  if (r.ech.length)
    out.push('ECH')
  if (r.params.some(p => p.key === 'ipv4hint' || p.key === 'ipv6hint'))
    out.push('hints')
  return out
}
