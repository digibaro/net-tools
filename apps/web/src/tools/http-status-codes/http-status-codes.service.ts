import type { StatusCode } from './http-status-codes.data'
import { STATUS_CODES } from './http-status-codes.data'

export type StatusClass = 'all' | '1' | '2' | '3' | '4' | '5'

export const rfcUrl = (s: Pick<StatusCode, 'rfc' | 'section'>) => `https://www.rfc-editor.org/rfc/rfc${s.rfc}#section-${s.section}`

export function filterCodes(query: string, cls: StatusClass, codes: StatusCode[] = STATUS_CODES): StatusCode[] {
  const q = query.trim().toLowerCase()
  return codes.filter(c =>
    (cls === 'all' || String(c.code)[0] === cls)
    && (!q || String(c.code).startsWith(q) || c.phrase.toLowerCase().includes(q) || c.description.toLowerCase().includes(q)))
}
