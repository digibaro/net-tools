import type { components } from '@/api/schema'

export type NdpRouter = components['schemas']['NdpRouter']
export interface NdpHost { ip: string, mac: string, router: boolean }

/** How hosts on this link get addresses, from the RA flags and prefixes (RFC 4861 / 8415). */
export type AddressMode = 'slaac' | 'dhcpv6' | 'slaac+dhcpv6-info' | 'slaac+dhcpv6' | 'none'
export function addressMode(r: NdpRouter): AddressMode {
  const slaac = r.prefixes.some(p => p.slaac)
  if (r.managed)
    return slaac ? 'slaac+dhcpv6' : 'dhcpv6'
  if (slaac)
    return r.other ? 'slaac+dhcpv6-info' : 'slaac'
  return 'none'
}

/** 86400 → "1 d", 1800 → "30 min" */
export function lifetime(s: number): string {
  if (s >= 0xFFFFFFFF)
    return '∞'
  if (s >= 86400 && s % 3600 === 0)
    return s % 86400 === 0 ? `${s / 86400} d` : `${s / 3600} h`
  if (s >= 3600 && s % 3600 === 0)
    return `${s / 3600} h`
  return s % 60 === 0 && s >= 60 ? `${s / 60} min` : `${s} s`
}
