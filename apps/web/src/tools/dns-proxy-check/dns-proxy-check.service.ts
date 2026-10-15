import type { DnsLookup } from '../dns-lookup/dns-lookup.service'
import type { EchoInfo } from '@/api/client'

/** Names with stable answers (anycast / single address) plus a random name that must not exist. */
export const TEST_NAMES = ['example.com', 'iana.org', 'one.one.one.one'] as const

export const randomName = () => `nettools-${crypto.getRandomValues(new Uint32Array(2)).join('')}.com`

export type DnsVerdict = 'match' | 'differ' | 'blocked' | 'nx-hijack' | 'ok-nx' | 'error'

const addresses = (l: DnsLookup) => l.answer.filter(r => r.type === 'A').map(r => r.data).sort()
const SINKHOLES = ['0.0.0.0', '127.0.0.1', '::', '::1']

/** Compares the server resolver with public DoH for one name. */
export function compareDns(server: DnsLookup | undefined, pub: DnsLookup | undefined, mustNotExist: boolean): DnsVerdict {
  if (!server)
    return 'error'
  const sa = addresses(server)
  if (mustNotExist)
    return sa.length ? 'nx-hijack' : 'ok-nx'
  if (!pub)
    return sa.length ? 'match' : 'error' // nothing to compare with; answered at least
  const pa = addresses(pub)
  if (pa.length && (!sa.length || sa.every(a => SINKHOLES.includes(a))))
    return 'blocked'
  if (sa.join() === pa.join())
    return 'match'
  return 'differ'
}

/** Headers proxies add or change. */
const PROXY_HEADERS = ['via', 'forwarded', 'x-forwarded-host', 'x-bluecoat-via', 'x-cache', 'x-squid-error', 'proxy-connection', 'x-proxy-id', 'client-ip', 'x-client-ip', 'cdn-loop', 'cf-connecting-ip', 'true-client-ip', 'x-original-forwarded-for']

export interface ProxyFinding { id: 'marker-missing' | 'marker-changed' | 'ua-changed' | 'proxy-header' | 'extra-hops' | 'clean', detail: string }

/** Compares what the browser sent with what the server received (our nginx adds X-Forwarded-For/-Proto and X-Real-IP). */
export function proxyFindings(echo: EchoInfo, marker: string, userAgent: string): ProxyFinding[] {
  const get = (n: string) => echo.headers.filter(h => h.name.toLowerCase() === n).map(h => h.value)
  const out: ProxyFinding[] = []
  const m = get('x-nettools-probe')
  if (!m.length)
    out.push({ id: 'marker-missing', detail: '' })
  else if (m[0] !== marker)
    out.push({ id: 'marker-changed', detail: m[0]! })
  if (echo.userAgent !== userAgent)
    out.push({ id: 'ua-changed', detail: echo.userAgent })
  for (const h of echo.headers) {
    if (PROXY_HEADERS.includes(h.name.toLowerCase()))
      out.push({ id: 'proxy-header', detail: `${h.name}: ${h.value}` })
  }
  const xff = get('x-forwarded-for').join(',').split(',').map(s => s.trim()).filter(Boolean)
  if (xff.length > 1)
    out.push({ id: 'extra-hops', detail: xff.join(', ') })
  return out.length ? out : [{ id: 'clean', detail: '' }]
}
