import type { components } from '@/api/schema'

export type TlsSession = components['schemas']['TlsSession']
export const STARTTLS = ['', 'smtp', 'imap', 'pop3', 'ftp', 'xmpp'] as const
export const DEFAULT_PORTS: Record<string, number> = { '': 443, 'smtp': 25, 'imap': 143, 'pop3': 110, 'ftp': 21, 'xmpp': 5222 }

/** Weak or outdated protocol versions. */
export const isLegacyVersion = (v: string) => v === 'TLS 1.0' || v === 'TLS 1.1' || v.startsWith('SSL')

export function expiryType(days: number): 'error' | 'warning' | 'success' {
  return days < 0 ? 'error' : days < 21 ? 'warning' : 'success'
}
