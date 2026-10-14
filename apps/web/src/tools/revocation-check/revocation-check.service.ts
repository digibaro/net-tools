import type { components } from '@/api/schema'

export type Revocation = components['schemas']['Revocation']

export const STATUS_TYPE = { good: 'success', revoked: 'error', unknown: 'warning' } as const

/** "Let's Encrypt has no OCSP" style hint when only CRLs exist. */
export const crlOnly = (r: Revocation) => r.crl.length > 0 && r.ocsp.every(o => o.source === 'stapled')
