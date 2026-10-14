import type { components } from '@/api/schema'

export type AbuseReport = components['schemas']['AbuseReport']
export { isIpAddress } from '../reverse-dns/reverse-dns.service'

export function scoreLevel(score: number): 'clean' | 'suspicious' | 'malicious' {
  if (score >= 75)
    return 'malicious'
  return score >= 25 ? 'suspicious' : 'clean'
}
