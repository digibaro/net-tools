import type { components } from '@/api/schema'

export type CtSearch = components['schemas']['CtSearch']

/** Subdomains below the queried domain, wildcards kept separate. */
export function splitNames(names: string[]): { hosts: string[], wildcards: string[] } {
  return { hosts: names.filter(n => !n.startsWith('*.')), wildcards: names.filter(n => n.startsWith('*.')) }
}
