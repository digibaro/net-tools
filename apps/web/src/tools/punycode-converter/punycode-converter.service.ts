import { toUnicode } from 'punycode/punycode.js'

export interface IdnRow {
  input: string
  ascii: string
  unicode: string
  /** Labels that differ between forms, i.e. are internationalized. */
  idnLabels: number
}

/**
 * Unicode → ASCII uses the WHATWG URL parser (UTS #46 mapping, as browsers resolve names);
 * ASCII → Unicode decodes each xn-- label.
 */
export function toAscii(domain: string): string | undefined {
  const d = domain.trim().replace(/\.$/, '')
  if (!d || /[\s/?#@:\\]/.test(d))
    return undefined
  try {
    const host = new URL(`http://${d}`).hostname
    return host && !host.startsWith('[') ? host : undefined
  }
  catch {
    return undefined
  }
}

export function convertDomain(input: string): IdnRow | undefined {
  const ascii = toAscii(input)
  if (ascii === undefined)
    return undefined
  let unicode: string
  try {
    unicode = toUnicode(ascii)
  }
  catch {
    return undefined
  }
  const idnLabels = ascii.split('.').filter(l => l.startsWith('xn--')).length
  return { input: input.trim(), ascii, unicode, idnLabels }
}

export function convertList(input: string): { rows: IdnRow[], invalid: string[] } {
  const rows: IdnRow[] = []
  const invalid: string[] = []
  for (const line of input.split(/[\n,]+/).map(s => s.trim()).filter(Boolean).slice(0, 1000)) {
    const r = convertDomain(line)
    if (r)
      rows.push(r)
    else
      invalid.push(line)
  }
  return { rows, invalid }
}
