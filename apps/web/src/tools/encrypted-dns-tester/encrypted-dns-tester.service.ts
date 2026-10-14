import type { components } from '@/api/schema'

export type DnsEncrypted = components['schemas']['DnsEncrypted']
export type EncryptedTransport = 'dot' | 'doh'

/** Validates the endpoint field for the chosen transport (the server validates fully). */
export function isValidEndpoint(value: string, transport: EncryptedTransport): boolean {
  const s = value.trim()
  if (!s)
    return false
  if (['cloudflare', 'google', 'quad9', 'opendns'].includes(s))
    return true
  if (transport === 'doh') {
    try {
      return new URL(s.includes('://') ? s : `https://${s}`).protocol === 'https:'
    }
    catch {
      return false
    }
  }
  return /^(?:\[[\da-f:.]+\]|[\w.-]+)(?::\d{1,5})?$/i.test(s)
}

export function daysUntil(iso: string, now = Date.now()): number {
  return Math.floor((new Date(iso).getTime() - now) / 86_400_000)
}
