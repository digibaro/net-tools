import type { components } from '@/api/schema'

export type RoaCoverage = components['schemas']['RoaCoverage']

/** Valid share of the checked prefixes (0–100). */
export function coverage(r: RoaCoverage): number {
  const checked = r.valid + r.invalid + r.notFound
  return checked ? Math.round(r.valid / checked * 1000) / 10 : 0
}

/** Minimal ROAs (RFC 9319) for prefixes without one: maxLength = the announced length. */
export function suggestions(r: RoaCoverage): string[] {
  return r.prefixes.filter(p => p.status === 'not-found').map(p => `${p.prefix} maxLength ${p.prefix.split('/')[1]} AS${r.asn}`)
}
