import type { components } from '@/api/schema'

export type DaneResult = components['schemas']['DaneResult']
export type DaneHost = DaneResult['hosts'][number]
export type DaneRecord = DaneHost['records'][number]

export const USAGES = ['PKIX-TA', 'PKIX-EE', 'DANE-TA', 'DANE-EE'] as const
export const SELECTORS = ['Cert', 'SPKI'] as const
export const MATCHING = ['Full', 'SHA2-256', 'SHA2-512'] as const

/** "3 1 1 (DANE-EE SPKI SHA2-256)" */
export function describeRecord(r: DaneRecord): string {
  return `${r.usage} ${r.selector} ${r.matchingType} (${USAGES[r.usage] ?? '?'} ${SELECTORS[r.selector] ?? '?'} ${MATCHING[r.matchingType] ?? '?'})`
}

/** Overall verdict for a domain: valid only when every host is valid. */
export function overall(r: DaneResult): DaneHost['verdict'] {
  const order: DaneHost['verdict'][] = ['mismatch', 'insecure', 'unreachable', 'no-tlsa', 'valid']
  return order.find(v => r.hosts.some(h => h.verdict === v)) ?? 'no-tlsa'
}
