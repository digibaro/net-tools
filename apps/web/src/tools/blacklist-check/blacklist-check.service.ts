import type { components } from '@/api/schema'

export type DnsblCheck = components['schemas']['DnsblCheck']
export type DnsblSummary = components['schemas']['DnsblSummary']

const ORDER = { listed: 0, error: 1, clean: 2 } as const

/** Listed first, then errors, then clean; alphabetical within each. */
export function sortChecks(list: DnsblCheck[]): DnsblCheck[] {
  return [...list].sort((a, b) => ORDER[a.status as keyof typeof ORDER] - ORDER[b.status as keyof typeof ORDER] || a.name.localeCompare(b.name))
}
