import type { components } from '@/api/schema'

export type HttpProtocols = components['schemas']['HttpProtocols']

/** Highest protocol the site supports from this server. */
export function best(p: HttpProtocols): 'HTTP/3' | 'HTTP/2' | 'HTTP/1.1' {
  if (p.http3.ok)
    return 'HTTP/3'
  return p.http2 ? 'HTTP/2' : 'HTTP/1.1'
}

export const isHttpsUrl = (s: string) => /^https:\/\/[^/\s]+/i.test(s.trim())
