import type { components } from '@/api/schema'

export type DnsCaa = components['schemas']['DnsCaa']

export type CaaVerdict
  = | { kind: 'any' }
    | { kind: 'blocked', reason: 'critical' | 'none-allowed' }
    | { kind: 'restricted', cas: string[] }

/** Issuer domain from an issue/issuewild value ("letsencrypt.org; validationmethods=dns-01"). */
export const issuerOf = (value: string) => value.split(';')[0]!.trim()

/** What the CAA set means for non-wildcard certificates (RFC 8659 §4.2). */
export function verdict(r: Pick<DnsCaa, 'relevantName' | 'issue' | 'unknownCritical'>): CaaVerdict {
  if (!r.relevantName)
    return { kind: 'any' }
  if (r.unknownCritical.length)
    return { kind: 'blocked', reason: 'critical' }
  const cas = [...new Set(r.issue.map(issuerOf).filter(Boolean))]
  if (!r.issue.length)
    return { kind: 'any' } // only issuewild/iodef present: no restriction for non-wildcards
  return cas.length ? { kind: 'restricted', cas } : { kind: 'blocked', reason: 'none-allowed' }
}

/** issuewild overrides issue for wildcard certificates; without it, issue applies. */
export function wildcardCas(r: Pick<DnsCaa, 'relevantName' | 'issue' | 'issueWild'>): string[] | 'any' {
  if (!r.relevantName)
    return 'any'
  const src = r.issueWild.length ? r.issueWild : r.issue
  if (!src.length)
    return 'any'
  return [...new Set(src.map(issuerOf).filter(Boolean))]
}
