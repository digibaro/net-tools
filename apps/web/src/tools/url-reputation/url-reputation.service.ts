import type { components } from '@/api/schema'

export type UrlReputation = components['schemas']['UrlReputation']
export type UrlVerdict = components['schemas']['UrlVerdict']

/** Accepts http(s) URLs or domain names with at least one dot (the server validates again). */
export function isUrlOrDomain(s: string): boolean {
  const v = s.trim()
  if (!v)
    return false
  try {
    const u = new URL(v.includes('://') ? v : `http://${v}`)
    return (u.protocol === 'http:' || u.protocol === 'https:') && u.hostname.includes('.') && !/^[\d.]+$/.test(u.hostname) && !u.hostname.startsWith('[')
  }
  catch {
    return false
  }
}

export const alertType = (v: UrlVerdict) => ({ malicious: 'error', suspicious: 'warning', clean: 'success' } as const)[v as 'malicious'] ?? 'info'
export const tagType = (v: UrlVerdict) => ({ malicious: 'error', suspicious: 'warning', clean: 'success', error: 'error' } as const)[v as 'malicious'] ?? 'default'
