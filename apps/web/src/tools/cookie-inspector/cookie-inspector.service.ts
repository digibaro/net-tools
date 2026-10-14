import type { components } from '@/api/schema'

export type Cookies = components['schemas']['Cookies']
export type CookieInfo = Cookies['cookies'][number]

/** Attribute summary: "Secure · HttpOnly · SameSite=Lax · Path=/". */
export function attributes(c: CookieInfo): string {
  return [c.secure ? 'Secure' : '', c.httpOnly ? 'HttpOnly' : '', c.sameSite ? `SameSite=${c.sameSite}` : '', c.partitioned ? 'Partitioned' : '', c.domain ? `Domain=${c.domain}` : '', c.path ? `Path=${c.path}` : ''].filter(Boolean).join(' · ')
}

export const severity = (c: CookieInfo) => (c.problems.length ? 'error' : c.warnings.some(w => w !== 'deleted') ? 'warning' : 'success')
