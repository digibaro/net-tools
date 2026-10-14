import { gunzipSync, unzipSync } from 'fflate'

/** Hard limits against decompression bombs. */
export const MAX_FILE_BYTES = 20 * 1024 * 1024
export const MAX_XML_BYTES = 20 * 1024 * 1024
export const MAX_ZIP_ENTRIES = 50
export const MAX_RECORDS = 20000

export interface AuthResult {
  domain: string
  result: string
  selector?: string
  scope?: string
}

export interface DmarcRecord {
  sourceIp: string
  count: number
  disposition: string
  dkim: string // policy_evaluated (aligned) result
  spf: string
  reasons: string[]
  headerFrom: string
  envelopeFrom: string
  dkimAuth: AuthResult[]
  spfAuth: AuthResult[]
  /** DMARC passes when aligned DKIM or aligned SPF passes. */
  pass: boolean
}

export interface DmarcReport {
  file: string
  orgName: string
  email: string
  reportId: string
  begin: Date | undefined
  end: Date | undefined
  policy: { domain: string, p: string, sp: string, pct: string, adkim: string, aspf: string }
  records: DmarcRecord[]
}

export class ReportError extends Error {
  constructor(public readonly key: 'tooLarge' | 'unsupported' | 'notDmarc' | 'doctype' | 'tooMany', public readonly file: string) {
    super(key)
  }
}

const text = (el: Element | null | undefined, sel: string) => el?.querySelector(sel)?.textContent?.trim() ?? ''

function epoch(s: string): Date | undefined {
  const n = Number(s)
  return s && Number.isFinite(n) ? new Date(n * 1000) : undefined
}

/** Parses one aggregate report XML (RFC 7489 appendix C). */
export function parseReportXml(xml: string, file = 'report.xml'): DmarcReport {
  // Reports never carry a DTD; refusing one rules out entity-expansion tricks.
  if (/<!DOCTYPE|<!ENTITY/i.test(xml))
    throw new ReportError('doctype', file)
  const doc = new DOMParser().parseFromString(xml, 'application/xml')
  const root = doc.documentElement
  if (doc.querySelector('parsererror') || root.localName !== 'feedback')
    throw new ReportError('notDmarc', file)
  const meta = root.querySelector('report_metadata')
  const pol = root.querySelector('policy_published')
  const records: DmarcRecord[] = []
  for (const rec of root.querySelectorAll(':scope > record')) {
    if (records.length >= MAX_RECORDS)
      break
    const row = rec.querySelector('row')
    const pe = row?.querySelector('policy_evaluated')
    const auth = (kind: 'dkim' | 'spf') => [...rec.querySelectorAll(`auth_results > ${kind}`)].map(a => ({ domain: text(a, 'domain'), result: text(a, 'result'), selector: text(a, 'selector') || undefined, scope: text(a, 'scope') || undefined }))
    const dkim = text(pe, 'dkim')
    const spf = text(pe, 'spf')
    records.push({
      sourceIp: text(row, 'source_ip'),
      count: Number(text(row, 'count')) || 0,
      disposition: text(pe, 'disposition') || 'none',
      dkim,
      spf,
      reasons: [...(pe?.querySelectorAll('reason') ?? [])].map(r => [text(r, 'type'), text(r, 'comment')].filter(Boolean).join(': ')),
      headerFrom: text(rec, 'identifiers > header_from'),
      envelopeFrom: text(rec, 'identifiers > envelope_from'),
      dkimAuth: auth('dkim'),
      spfAuth: auth('spf'),
      pass: dkim === 'pass' || spf === 'pass',
    })
  }
  return {
    file,
    orgName: text(meta, 'org_name'),
    email: text(meta, 'email'),
    reportId: text(meta, 'report_id'),
    begin: epoch(text(meta, 'date_range > begin')),
    end: epoch(text(meta, 'date_range > end')),
    policy: { domain: text(pol, 'domain'), p: text(pol, 'p'), sp: text(pol, 'sp'), pct: text(pol, 'pct'), adkim: text(pol, 'adkim') || 'r', aspf: text(pol, 'aspf') || 'r' },
    records,
  }
}

const utf8 = (b: Uint8Array) => new TextDecoder().decode(b)

/** Gzip stores the uncompressed size (mod 2^32) in its last 4 bytes. */
function gzipSize(b: Uint8Array): number {
  const n = b.length
  return n < 4 ? 0 : (b[n - 4]! | (b[n - 3]! << 8) | (b[n - 2]! << 16) | (b[n - 1]! << 24)) >>> 0
}

/** Accepts .xml, .gz (gzip) and .zip (one or more XML reports); sniffed by magic bytes. */
export function readReports(name: string, bytes: Uint8Array): DmarcReport[] {
  if (bytes.length > MAX_FILE_BYTES)
    throw new ReportError('tooLarge', name)
  if (bytes[0] === 0x1F && bytes[1] === 0x8B) {
    if (gzipSize(bytes) > MAX_XML_BYTES)
      throw new ReportError('tooLarge', name)
    return [parseReportXml(utf8(gunzipSync(bytes)), name)]
  }
  if (bytes[0] === 0x50 && bytes[1] === 0x4B) {
    let entries = 0
    let total = 0
    const files = unzipSync(bytes, {
      filter: (f) => {
        if (!/\.xml$/i.test(f.name))
          return false
        entries++
        total += f.originalSize
        if (entries > MAX_ZIP_ENTRIES)
          throw new ReportError('tooMany', name)
        if (f.originalSize > MAX_XML_BYTES || total > MAX_XML_BYTES)
          throw new ReportError('tooLarge', name)
        return true
      },
    })
    const names = Object.keys(files)
    if (!names.length)
      throw new ReportError('notDmarc', name)
    return names.map(n => parseReportXml(utf8(files[n]!), `${name} › ${n}`))
  }
  const s = utf8(bytes)
  if (!s.trimStart().startsWith('<'))
    throw new ReportError('unsupported', name)
  return [parseReportXml(s, name)]
}

export interface SourceRow {
  sourceIp: string
  count: number
  pass: number
  fail: number
  headerFrom: string[]
  dispositions: Record<string, number>
  dkim: string[]
  spf: string[]
}

export interface Summary {
  messages: number
  pass: number
  fail: number
  passRate: number
  dispositions: Record<string, number>
  reporters: string[]
  domains: string[]
  begin?: Date
  end?: Date
  sources: SourceRow[]
}

function add<T>(xs: T[], x: T): void {
  if (!xs.includes(x))
    xs.push(x)
}

/** Aggregates all reports by source IP, worst offenders (most failing messages) first. */
export function summarize(reports: DmarcReport[]): Summary {
  const sources = new Map<string, SourceRow>()
  const s: Summary = { messages: 0, pass: 0, fail: 0, passRate: 0, dispositions: {}, reporters: [], domains: [], sources: [] }
  for (const r of reports) {
    add(s.reporters, r.orgName || r.email)
    if (r.policy.domain)
      add(s.domains, r.policy.domain)
    if (r.begin && (!s.begin || r.begin < s.begin))
      s.begin = r.begin
    if (r.end && (!s.end || r.end > s.end))
      s.end = r.end
    for (const rec of r.records) {
      s.messages += rec.count
      s[rec.pass ? 'pass' : 'fail'] += rec.count
      s.dispositions[rec.disposition] = (s.dispositions[rec.disposition] ?? 0) + rec.count
      const row = sources.get(rec.sourceIp) ?? { sourceIp: rec.sourceIp, count: 0, pass: 0, fail: 0, headerFrom: [], dispositions: {}, dkim: [], spf: [] }
      row.count += rec.count
      row[rec.pass ? 'pass' : 'fail'] += rec.count
      row.dispositions[rec.disposition] = (row.dispositions[rec.disposition] ?? 0) + rec.count
      if (rec.headerFrom)
        add(row.headerFrom, rec.headerFrom)
      for (const a of rec.dkimAuth) add(row.dkim, `${a.domain}${a.selector ? ` (${a.selector})` : ''}: ${a.result}`)
      for (const a of rec.spfAuth) add(row.spf, `${a.domain}: ${a.result}`)
      sources.set(rec.sourceIp, row)
    }
  }
  s.passRate = s.messages ? s.pass / s.messages : 0
  s.sources = [...sources.values()].sort((a, b) => b.fail - a.fail || b.count - a.count)
  return s
}
