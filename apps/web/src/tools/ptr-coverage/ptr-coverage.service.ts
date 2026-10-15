import type { components } from '@/api/schema'
import { parseCidr, parseIP } from '@/utils/ip'

export type PtrScanResult = components['schemas']['PtrScanResult']
export type PtrStatus = 'confirmed' | 'mismatch' | 'none' | 'error'

/** At most 256 addresses: IPv4 /24 or longer, IPv6 /120 or longer; a bare address means its /24 or /120. */
export function isScanPrefix(s: string): boolean {
  const v = s.trim()
  const c = parseCidr(v, { allowBare: false })
  if (c)
    return (c.version === 4 ? 32 : 128) - c.prefix <= 8
  return !!parseIP(v)
}

export function status(r: PtrScanResult['rows'][number]): PtrStatus {
  if (r.error)
    return 'error'
  if (!r.names.length)
    return 'none'
  return r.confirmed ? 'confirmed' : 'mismatch'
}
