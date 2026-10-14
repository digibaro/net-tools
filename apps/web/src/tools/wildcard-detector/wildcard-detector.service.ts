import type { components } from '@/api/schema'

export type DnsWildcard = components['schemas']['DnsWildcard']

/** Distinct answers across probes, with the probe name replaced by "*". */
export function wildcardAnswers(r: DnsWildcard): string[] {
  const out = new Set<string>()
  for (const p of r.probes) {
    for (const a of p.answers)
      out.add(`*.${r.domain}. ${a.ttl} ${a.type} ${a.data}`)
  }
  return [...out]
}
