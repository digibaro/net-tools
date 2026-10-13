import type { components } from '@/api/schema'

export type Cors = components['schemas']['Cors']

/** An Origin header value: scheme://host[:port] with nothing else. */
export function isValidOrigin(input: string): boolean {
  try {
    const u = new URL(input.trim())
    return (u.protocol === 'http:' || u.protocol === 'https:') && (u.pathname === '/' || u.pathname === '') && !u.search && !u.hash && !u.username && `${u.protocol}//${u.host}` === input.trim().replace(/\/$/, '')
  }
  catch {
    return false
  }
}

export const isValidHeaderList = (s: string) => !s.trim() || s.split(',').every(h => /^[!#$%&'*+\-.^\w`|~]+$/.test(h.trim()))
