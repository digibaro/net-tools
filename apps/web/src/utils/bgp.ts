import { parseCidr } from './ip'

/** "AS13335", "as 13335" or "13335" → 13335; undefined when not a 32-bit AS number. */
export function parseAsn(input: string): number | undefined {
  const m = /^(?:as\s?)?(\d{1,10})$/i.exec(input.trim())
  if (!m)
    return undefined
  const n = Number(m[1])
  return n <= 4294967295 ? n : undefined
}

/** A prefix (1.1.1.0/24) or a single address; the server normalizes and rejects unrouted space. */
export const isPrefixOrIp = (input: string) => !!parseCidr(input.trim())

export const asnPath = (asn: number) => `/asn-info?asn=AS${asn}`
export const prefixPath = (prefix: string) => `/prefix-status?prefix=${encodeURIComponent(prefix)}`
