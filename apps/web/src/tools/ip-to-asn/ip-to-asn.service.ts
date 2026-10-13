import type { components } from '@/api/schema'

export type AsnLookup = components['schemas']['AsnLookup']
export { isIpAddress } from '../reverse-dns/reverse-dns.service'

export const RIR_NAMES: Record<string, string> = { arin: 'ARIN', ripencc: 'RIPE NCC', apnic: 'APNIC', lacnic: 'LACNIC', afrinic: 'AFRINIC' }
export const rirName = (r: string) => RIR_NAMES[r.toLowerCase()] ?? r.toUpperCase()
