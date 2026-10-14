import type { components } from '@/api/schema'

export type DnssecResult = components['schemas']['DnssecResult']
export type DnssecStep = components['schemas']['DnssecStep']

export const STATUS_TYPE = { secure: 'success', insecure: 'warning', bogus: 'error', indeterminate: 'default' } as const

export const TYPES = ['A', 'AAAA', 'MX', 'TXT', 'NS', 'SOA', 'CNAME', 'CAA', 'DNSKEY', 'DS', 'HTTPS', 'TLSA', 'SRV']

/** One line per key: "KSK 20326 RSASHA256 ✓". */
export const keyLine = (k: DnssecStep['keys'][number]) => `${k.role} ${k.tag} ${k.algorithm}${k.trusted ? ' ✓' : ''}`
