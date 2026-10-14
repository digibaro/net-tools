import { isValidHost } from './host'

/** Accepts a domain, an IPv4/IPv6 address or an AS number (AS13335 or 13335). */
export function isRegistryQuery(raw: string): boolean {
  const s = raw.trim()
  if (/^(?:as)?\d{1,10}$/i.test(s))
    return true
  return isValidHost(s.replace(/^\[|\]$/g, '')) && (s.includes('.') || s.includes(':'))
}
