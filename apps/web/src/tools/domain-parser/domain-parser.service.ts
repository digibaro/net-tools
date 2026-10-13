import { parse } from 'tldts'

export interface DomainParts {
  hostname: string
  subdomain: string
  domain: string
  domainWithoutSuffix: string
  publicSuffix: string
  suffixType: 'icann' | 'private' | 'unknown'
}

export interface DomainAnalysis {
  isIp: boolean
  /** With the full Public Suffix List (ICANN + private, e.g. github.io). */
  withPrivate: DomainParts
  /** ICANN section only. */
  icannOnly: DomainParts
}

function parts(input: string, allowPrivateDomains: boolean): DomainParts | undefined {
  const r = parse(input, { allowPrivateDomains, extractHostname: true, validateHostname: true })
  if (!r.hostname || r.isIp)
    return undefined
  return {
    hostname: r.hostname,
    subdomain: r.subdomain ?? '',
    domain: r.domain ?? '',
    domainWithoutSuffix: r.domainWithoutSuffix ?? '',
    publicSuffix: r.publicSuffix ?? '',
    suffixType: r.isPrivate ? 'private' : r.isIcann ? 'icann' : 'unknown',
  }
}

/** Accepts a hostname or a full URL. */
export function analyzeDomain(input: string): DomainAnalysis | undefined {
  const s = input.trim()
  if (!s)
    return undefined
  if (parse(s, { extractHostname: true }).isIp)
    return { isIp: true, withPrivate: emptyParts(s), icannOnly: emptyParts(s) }
  const withPrivate = parts(s, true)
  const icannOnly = parts(s, false)
  return withPrivate && icannOnly ? { isIp: false, withPrivate, icannOnly } : undefined
}

function emptyParts(hostname: string): DomainParts {
  return { hostname, subdomain: '', domain: '', domainWithoutSuffix: '', publicSuffix: '', suffixType: 'unknown' }
}
