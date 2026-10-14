export interface Header { name: string, value: string }

export const MAX_INPUT = 512 * 1024

/** Splits a raw message (or header block) into unfolded headers; stops at the first empty line. */
export function parseHeaders(raw: string): Header[] {
  const out: Header[] = []
  for (const line of raw.slice(0, MAX_INPUT).replace(/\r\n?/g, '\n').split('\n')) {
    if (line.trim() === '') {
      if (out.length)
        break
      continue
    }
    if (/^[ \t]/.test(line) && out.length) {
      out.at(-1)!.value += ` ${line.trim()}`
      continue
    }
    const colon = line.indexOf(':')
    if (colon < 1)
      continue
    out.push({ name: line.slice(0, colon).trim(), value: line.slice(colon + 1).trim() })
  }
  return out
}

/** Parses an RFC 5322 date, ignoring trailing comments such as "(UTC)". */
export function parseDate(s: string): Date | undefined {
  const cleaned = s.replace(/\([^)]*\)/g, '').replace(/\s+/g, ' ').trim()
  const d = new Date(cleaned)
  return Number.isNaN(d.getTime()) ? undefined : d
}

export interface Hop {
  index: number
  from: string
  by: string
  with: string
  date?: Date
  /** Seconds since the previous hop; undefined for the first hop or missing dates. */
  delay?: number
  raw: string
}

function clause(value: string, word: string): string {
  const m = new RegExp(`(?:^|\\s)${word}\\s+(\\S+(?:\\s+\\([^)]*\\))?)`, 'i').exec(value)
  return m ? m[1]!.replace(/\s+/g, ' ').trim() : ''
}

/** Received headers in chronological order (the bottom header is the first hop). */
export function hops(headers: Header[]): Hop[] {
  const received = headers.filter(h => h.name.toLowerCase() === 'received').reverse()
  const out: Hop[] = []
  received.forEach((h, i) => {
    const semi = h.value.lastIndexOf(';')
    const body = semi === -1 ? h.value : h.value.slice(0, semi)
    const date = semi === -1 ? undefined : parseDate(h.value.slice(semi + 1))
    const prev = out.at(-1)?.date
    out.push({
      index: i + 1,
      from: clause(body, 'from'),
      by: clause(body, 'by'),
      with: clause(body, 'with').split(' ')[0] ?? '',
      date,
      delay: date && prev ? Math.round((date.getTime() - prev.getTime()) / 1000) : undefined,
      raw: h.value,
    })
  })
  return out
}

export interface AuthResult { method: string, result: string, detail: string }

/** Parses Authentication-Results (RFC 8601) into method=result entries. */
export function authResults(headers: Header[]): AuthResult[] {
  const out: AuthResult[] = []
  for (const h of headers.filter(h => h.name.toLowerCase() === 'authentication-results' || h.name.toLowerCase() === 'arc-authentication-results')) {
    if (h.name.toLowerCase() === 'arc-authentication-results' && out.length)
      continue
    const parts = h.value.replace(/\([^)]*\)/g, '').split(';').slice(1)
    for (const p of parts) {
      const eq = p.indexOf('=')
      const method = p.slice(0, eq).trim()
      const rest = p.slice(eq + 1).trim()
      const result = /^[\w-]+/.exec(rest)?.[0]
      if (eq > 0 && /^[\w.-]+$/.test(method) && result)
        out.push({ method: method.toLowerCase(), result: result.toLowerCase(), detail: rest.slice(result.length).trim().replace(/\s+/g, ' ') })
    }
  }
  return out
}

export interface DkimSignature { domain: string, selector: string, algorithm: string }

export function dkimSignatures(headers: Header[]): DkimSignature[] {
  return headers.filter(h => h.name.toLowerCase() === 'dkim-signature').map((h) => {
    const tags = Object.fromEntries(h.value.split(';').map(t => t.split('=').map(s => s.trim())).filter(t => t.length >= 2).map(([k, ...v]) => [k!.toLowerCase(), v.join('=').replace(/\s+/g, '')]))
    return { domain: tags.d ?? '', selector: tags.s ?? '', algorithm: tags.a ?? '' }
  })
}

export const SUMMARY_HEADERS = ['From', 'To', 'Cc', 'Subject', 'Date', 'Message-ID', 'Return-Path', 'Reply-To', 'Delivered-To', 'X-Mailer', 'User-Agent']

export function summary(headers: Header[]): Header[] {
  return SUMMARY_HEADERS.flatMap((name) => {
    const h = headers.find(x => x.name.toLowerCase() === name.toLowerCase())
    return h ? [{ name, value: h.value }] : []
  })
}

/** Human duration for a delay in seconds ("2 min 5 s"); negative delays mean clock skew. */
export function formatDelay(s: number): string {
  const sign = s < 0 ? '−' : ''
  let v = Math.abs(s)
  const h = Math.floor(v / 3600)
  v -= h * 3600
  const m = Math.floor(v / 60)
  v -= m * 60
  return sign + [h ? `${h} h` : '', m ? `${m} min` : '', v || (!h && !m) ? `${v} s` : ''].filter(Boolean).join(' ')
}
