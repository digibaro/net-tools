import type { components } from '@/api/schema'

export type RdapRecord = components['schemas']['RdapRecord']
export { isRegistryQuery } from '@/utils/registry-query'

/** Formats an RDAP date (RFC 3339) as YYYY-MM-DD; returns the input if unparsable. */
export function day(s: string): string {
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? s : d.toISOString().slice(0, 10)
}
