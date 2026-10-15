import type { components } from '@/api/schema'

export type ChainResult = components['schemas']['ChainResult']

export type Verdict = 'ok' | 'unordered' | 'incomplete' | 'untrusted'
export function verdict(r: ChainResult): Verdict {
  if (!r.trusted)
    return 'untrusted'
  if (!r.complete)
    return 'incomplete'
  return r.ordered ? 'ok' : 'unordered'
}

export const countCerts = (pem: string) => (pem.match(/-----BEGIN CERTIFICATE-----/g) ?? []).length
