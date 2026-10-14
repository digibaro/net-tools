import type { components } from '@/api/schema'

export type EmailDmarc = components['schemas']['EmailDmarc']

export function strength(policy: string, pct: number): 'none' | 'weak' | 'strong' {
  if (policy === 'reject' || (policy === 'quarantine' && pct === 100))
    return 'strong'
  return policy === 'quarantine' ? 'weak' : 'none'
}
