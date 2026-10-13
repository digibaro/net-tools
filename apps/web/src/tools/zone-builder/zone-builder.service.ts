import { formatIPv4, formatIPv6, parseCidr, parseIPv4, parseIPv6 } from '@/utils/ip'

export const RECORD_KINDS = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'SPF', 'SRV', 'CAA', 'NS', 'PTR'] as const
export type RecordKind = typeof RECORD_KINDS[number]

export type FieldError = 'name' | 'ttl' | 'address' | 'target' | 'number' | 'text' | 'value' | 'service'
export type Built = { line: string } | { error: FieldError }

const LABEL = /^(?:\*|[\w-]{1,63})$/
const encoder = new TextEncoder()

/** Owner or target name: "@", relative ("www"), or absolute with a trailing dot. Underscores allowed (_dmarc, _sip). */
export function isValidName(name: string, { allowRoot = false } = {}): boolean {
  const s = name.trim()
  if (s === '@' || (allowRoot && s === '.'))
    return true
  const body = s.endsWith('.') ? s.slice(0, -1) : s
  if (!body || encoder.encode(body).length > 253)
    return false
  const labels = body.split('.')
  return labels.every((l, i) => LABEL.test(l) && !l.startsWith('-') && !l.endsWith('-') && (l !== '*' || i === 0))
}

/** Escapes one TXT character-string (≤ 255 bytes) for zone files: quotes, backslashes and non-printable bytes as \DDD. */
function quoteBytes(bytes: Uint8Array): string {
  let out = '"'
  for (const b of bytes) {
    if (b === 0x22 || b === 0x5C)
      out += `\\${String.fromCharCode(b)}`
    else if (b < 0x20 || b > 0x7E)
      out += `\\${b.toString().padStart(3, '0')}`
    else
      out += String.fromCharCode(b)
  }
  return `${out}"`
}

/** Splits text into 255-byte character-strings (RFC 1035 §3.3.14); receivers concatenate them. */
export function txtStrings(text: string): string[] {
  const bytes = encoder.encode(text)
  if (bytes.length === 0)
    return ['""']
  const out: string[] = []
  for (let i = 0; i < bytes.length; i += 255)
    out.push(quoteBytes(bytes.subarray(i, i + 255)))
  return out
}

const u16 = (n: number) => Number.isInteger(n) && n >= 0 && n <= 65535

export interface Common { name: string, ttl?: number }

export type RecordInput
  = | Common & { kind: 'A' | 'AAAA', address: string }
    | Common & { kind: 'CNAME' | 'NS' | 'PTR', target: string }
    | Common & { kind: 'MX', preference: number, target: string }
    | Common & { kind: 'TXT', text: string }
    | Common & { kind: 'SPF', spf: SpfInput }
    | Common & { kind: 'SRV', service: string, protocol: string, priority: number, weight: number, port: number, target: string }
    | Common & { kind: 'CAA', critical: boolean, tag: 'issue' | 'issuewild' | 'iodef' | 'issuemail', value: string }

function line(name: string, ttl: number | undefined, type: string, rdata: string): string {
  return [name.trim(), ttl === undefined ? '' : String(ttl), 'IN', type, rdata].filter(Boolean).join('\t')
}

/** Builds one zone-file line, or reports the first invalid field. */
export function buildRecord(r: RecordInput): Built {
  if (r.kind !== 'SRV' && !isValidName(r.name))
    return { error: 'name' }
  if (r.ttl !== undefined && !(Number.isInteger(r.ttl) && r.ttl >= 0 && r.ttl <= 2147483647))
    return { error: 'ttl' }
  switch (r.kind) {
    case 'A': {
      const v = parseIPv4(r.address.trim())
      return v === undefined ? { error: 'address' } : { line: line(r.name, r.ttl, 'A', formatIPv4(v)) }
    }
    case 'AAAA': {
      const v = parseIPv6(r.address.trim())
      return v === undefined ? { error: 'address' } : { line: line(r.name, r.ttl, 'AAAA', formatIPv6(v)) }
    }
    case 'CNAME':
    case 'NS':
    case 'PTR':
      return isValidName(r.target) ? { line: line(r.name, r.ttl, r.kind, r.target.trim()) } : { error: 'target' }
    case 'MX':
      if (!u16(r.preference))
        return { error: 'number' }
      return isValidName(r.target, { allowRoot: true }) ? { line: line(r.name, r.ttl, 'MX', `${r.preference} ${r.target.trim()}`) } : { error: 'target' }
    case 'TXT':
      return { line: line(r.name, r.ttl, 'TXT', txtStrings(r.text).join(' ')) }
    case 'SPF': {
      const spf = buildSpf(r.spf)
      return 'error' in spf ? { error: 'value' } : { line: line(r.name, r.ttl, 'TXT', txtStrings(spf.record).join(' ')) }
    }
    case 'SRV': {
      const svc = r.service.trim().replace(/^_?/, '_')
      const proto = r.protocol.trim().replace(/^_?/, '_')
      if (!/^_[a-z\d-]{1,62}$/i.test(svc) || !/^_[a-z\d-]{1,62}$/i.test(proto))
        return { error: 'service' }
      const base = r.name.trim()
      const owner = base === '@' || base === '' ? `${svc}.${proto}` : `${svc}.${proto}.${base}`
      if (!isValidName(owner))
        return { error: 'name' }
      if (![r.priority, r.weight, r.port].every(u16))
        return { error: 'number' }
      return isValidName(r.target, { allowRoot: true }) ? { line: line(owner, r.ttl, 'SRV', `${r.priority} ${r.weight} ${r.port} ${r.target.trim()}`) } : { error: 'target' }
    }
    case 'CAA': {
      const v = r.value.trim()
      if (!v || (r.tag === 'iodef' && !/^(?:mailto:|https?:\/\/)\S+$/.test(v)))
        return { error: 'value' }
      return { line: line(r.name, r.ttl, 'CAA', `${r.critical ? 128 : 0} ${r.tag} ${txtStrings(v)[0]}`) }
    }
  }
}

export interface SpfInput {
  mx: boolean
  a: boolean
  includes: string[]
  ip4: string[]
  ip6: string[]
  all: '-' | '~' | '?'
}

/** Builds "v=spf1 …" and counts DNS-querying mechanisms (RFC 7208 limit: 10). */
export function buildSpf(s: SpfInput): { record: string, lookups: number } | { error: string } {
  const terms = ['v=spf1']
  for (const ip of s.ip4) {
    const c = parseCidr(ip)
    if (!c || c.version !== 4)
      return { error: ip }
    terms.push(`ip4:${ip.trim()}`)
  }
  for (const ip of s.ip6) {
    const c = parseCidr(ip)
    if (!c || c.version !== 6)
      return { error: ip }
    terms.push(`ip6:${ip.trim()}`)
  }
  if (s.a)
    terms.push('a')
  if (s.mx)
    terms.push('mx')
  for (const inc of s.includes) {
    if (!isValidName(inc) || inc.trim() === '@')
      return { error: inc }
    terms.push(`include:${inc.trim().replace(/\.$/, '')}`)
  }
  terms.push(`${s.all}all`)
  return { record: terms.join(' '), lookups: (s.a ? 1 : 0) + (s.mx ? 1 : 0) + s.includes.length }
}

export interface Soa { origin: string, ttl: number, primary: string, email: string, serial: number, refresh: number, retry: number, expire: number, minimum: number }

/** Today's serial in YYYYMMDDnn form. */
export function dateSerial(now = new Date(), revision = 1): number {
  const y = now.getUTCFullYear()
  const m = String(now.getUTCMonth() + 1).padStart(2, '0')
  const d = String(now.getUTCDate()).padStart(2, '0')
  return Number(`${y}${m}${d}${String(revision).padStart(2, '0')}`)
}

/** hostmaster@example.com → hostmaster.example.com. (dots in the local part escaped). */
export function rname(email: string): string | undefined {
  const at = email.trim().lastIndexOf('@')
  if (at < 1)
    return undefined
  const local = email.trim().slice(0, at).replace(/\./g, '\\.')
  const domain = email.trim().slice(at + 1).replace(/\.$/, '')
  return isValidName(`${domain}.`) ? `${local}.${domain}.` : undefined
}

export function zoneHeader(z: Soa): string | undefined {
  const origin = z.origin.trim().replace(/\.?$/, '.')
  const mbox = rname(z.email)
  if (!isValidName(origin) || !isValidName(z.primary) || !mbox)
    return undefined
  return [
    `$ORIGIN ${origin}`,
    `$TTL ${z.ttl}`,
    `@\tIN\tSOA\t${z.primary.trim()} ${mbox} (`,
    `\t\t${z.serial}\t; serial`,
    `\t\t${z.refresh}\t; refresh`,
    `\t\t${z.retry}\t; retry`,
    `\t\t${z.expire}\t; expire`,
    `\t\t${z.minimum} )\t; negative-caching TTL`,
  ].join('\n')
}
