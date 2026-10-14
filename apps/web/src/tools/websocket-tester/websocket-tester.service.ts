export const MAX_LOG = 500

export interface LogEntry {
  dir: 'in' | 'out' | 'sys'
  at: number
  text: string
  bytes: number
}

export function isWsUrl(s: string): boolean {
  try {
    const u = new URL(s.trim())
    return (u.protocol === 'ws:' || u.protocol === 'wss:') && !!u.hostname
  }
  catch {
    return false
  }
}

export const parseProtocols = (s: string) => s.split(',').map(p => p.trim()).filter(Boolean)

/** RFC 6455 §7.4 close codes, as i18n keys. */
const CLOSE_CODES: Record<number, string> = { 1000: 'normal', 1001: 'goingAway', 1002: 'protocolError', 1003: 'unsupported', 1005: 'noStatus', 1006: 'abnormal', 1007: 'invalidData', 1008: 'policy', 1009: 'tooBig', 1010: 'extension', 1011: 'serverError', 1012: 'restart', 1013: 'tryAgain', 1015: 'tls' }

export const closeKey = (code: number) => CLOSE_CODES[code] ?? 'other'

/** Hex preview of the first bytes of a binary frame. */
export function hexPreview(buf: ArrayBuffer, max = 32): string {
  const b = new Uint8Array(buf, 0, Math.min(max, buf.byteLength))
  return [...b].map(x => x.toString(16).padStart(2, '0')).join(' ') + (buf.byteLength > max ? ' …' : '')
}

export function pushLog(log: LogEntry[], e: LogEntry): LogEntry[] {
  const next = [...log, e]
  return next.length > MAX_LOG ? next.slice(next.length - MAX_LOG) : next
}
