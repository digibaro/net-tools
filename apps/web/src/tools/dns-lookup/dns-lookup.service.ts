import type { components } from '@/api/schema'
import { toAscii } from '../punycode-converter/punycode-converter.service'

export type DnsLookup = components['schemas']['DnsLookup']
export type DnsRecord = components['schemas']['DnsRecord']
export type RecordType = components['schemas']['DnsRecordType']

export const RECORD_TYPES: RecordType[] = ['A', 'AAAA', 'CNAME', 'MX', 'NS', 'TXT', 'SOA', 'SRV', 'CAA', 'PTR', 'DS', 'DNSKEY', 'HTTPS', 'SVCB', 'TLSA', 'NAPTR', 'ANY']
export const RESOLVER_PRESETS = ['system', 'cloudflare', 'google', 'quad9', 'opendns'] as const
export const TRANSPORTS = ['udp', 'tcp', 'dot', 'doh'] as const

const TYPE_NUMBERS: Record<number, string> = { 1: 'A', 2: 'NS', 5: 'CNAME', 6: 'SOA', 12: 'PTR', 15: 'MX', 16: 'TXT', 28: 'AAAA', 33: 'SRV', 35: 'NAPTR', 43: 'DS', 46: 'RRSIG', 47: 'NSEC', 48: 'DNSKEY', 50: 'NSEC3', 52: 'TLSA', 64: 'SVCB', 65: 'HTTPS', 257: 'CAA' }
const RCODES = ['NOERROR', 'FORMERR', 'SERVFAIL', 'NXDOMAIN', 'NOTIMP', 'REFUSED']

/** A domain name we can query (IDN converted); IPv4/IPv6 addresses are allowed for PTR. */
export function isValidQueryName(name: string): boolean {
  const s = name.trim()
  if (!s)
    return false
  if (/^[\d.]+$/.test(s) || /^[\da-f:.]+$/i.test(s))
    return true // the server validates addresses (PTR)
  return toAscii(s) !== undefined
}

export const DOH_JSON = {
  cloudflare: (name: string, type: string) => `https://cloudflare-dns.com/dns-query?name=${encodeURIComponent(name)}&type=${type}`,
  google: (name: string, type: string) => `https://dns.google/resolve?name=${encodeURIComponent(name)}&type=${type}`,
} as const
export type DohProvider = keyof typeof DOH_JSON

interface DohJsonRecord { name: string, type: number, TTL: number, data: string }
interface DohJson {
  Status: number
  TC?: boolean
  RD?: boolean
  RA?: boolean
  AD?: boolean
  CD?: boolean
  Answer?: DohJsonRecord[]
  Authority?: DohJsonRecord[]
  Additional?: DohJsonRecord[]
}

function records(list: DohJsonRecord[] | undefined): DnsRecord[] {
  return (list ?? []).map(r => ({ name: r.name.endsWith('.') ? r.name : `${r.name}.`, type: TYPE_NUMBERS[r.type] ?? `TYPE${r.type}`, ttl: r.TTL, class: 'IN', data: r.data }))
}

/** Maps a DoH JSON answer (Cloudflare / Google format) to the server's DnsLookup shape. */
export function fromDohJson(json: DohJson, name: string, type: string, provider: DohProvider, rttMs: number): DnsLookup {
  return {
    name,
    type,
    resolver: provider === 'cloudflare' ? 'Cloudflare DoH (from your browser)' : 'Google DoH (from your browser)',
    transport: 'doh',
    rcode: RCODES[json.Status] ?? String(json.Status),
    flags: { aa: false, tc: !!json.TC, rd: json.RD ?? true, ra: !!json.RA, ad: !!json.AD, cd: !!json.CD },
    answer: records(json.Answer),
    authority: records(json.Authority),
    additional: records(json.Additional),
    rttMs: Math.round(rttMs * 1000) / 1000,
  }
}

export async function lookupInBrowser(name: string, type: string, provider: DohProvider, fetchFn: typeof fetch = fetch): Promise<DnsLookup> {
  const ascii = toAscii(name) ?? name.trim()
  const start = performance.now()
  const res = await fetchFn(DOH_JSON[provider](ascii, type), { headers: { Accept: 'application/dns-json' }, referrerPolicy: 'no-referrer' })
  if (!res.ok)
    throw new Error(`DoH request failed (HTTP ${res.status})`)
  const json = await res.json() as DohJson
  return fromDohJson(json, ascii, type, provider, performance.now() - start)
}
