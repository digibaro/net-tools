import type { components } from '@/api/schema'

export type DomainExpiry = components['schemas']['DomainExpiry']
export { day } from '../rdap-lookup/rdap-lookup.service'

export function urgency(daysLeft: number | null): 'expired' | 'soon' | 'ok' | 'unknown' {
  if (daysLeft === null)
    return 'unknown'
  if (daysLeft < 0)
    return 'expired'
  return daysLeft <= 30 ? 'soon' : 'ok'
}
