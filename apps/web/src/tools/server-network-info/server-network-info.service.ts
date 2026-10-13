import type { components } from '@/api/schema'

export type ServerNetinfo = components['schemas']['ServerNetinfo']

/** Interfaces worth showing first: up, non-loopback, with addresses. */
export function sortInterfaces(list: ServerNetinfo['interfaces']): ServerNetinfo['interfaces'] {
  const score = (i: ServerNetinfo['interfaces'][number]) => (i.flags.includes('loopback') ? 2 : 0) + (i.flags.includes('up') ? 0 : 1)
  return [...list].sort((a, b) => score(a) - score(b) || a.name.localeCompare(b.name))
}
