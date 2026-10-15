import type { DnsLookup } from '../dns-lookup/dns-lookup.service'
import type { EchoInfo } from '@/api/client'
import { describe, expect, it } from 'vitest'
import { compareDns, proxyFindings } from './dns-proxy-check.service'

const l = (...ips: string[]) => ({ answer: ips.map(data => ({ type: 'A', data })) }) as unknown as DnsLookup

describe('dns-proxy-check', () => {
  it('compares DNS answers', () => {
    expect(compareDns(l('1.2.3.4'), l('1.2.3.4'), false)).toBe('match')
    expect(compareDns(l('1.2.3.5'), l('1.2.3.4'), false)).toBe('differ')
    expect(compareDns(l('0.0.0.0'), l('1.2.3.4'), false)).toBe('blocked')
    expect(compareDns(l(), l('1.2.3.4'), false)).toBe('blocked')
    expect(compareDns(l('9.9.9.9'), l(), true)).toBe('nx-hijack')
    expect(compareDns(l(), l(), true)).toBe('ok-nx')
    expect(compareDns(undefined, l('1.2.3.4'), false)).toBe('error')
  })
  it('spots proxies', () => {
    const echo = (headers: [string, string][], userAgent = 'UA') => ({ userAgent, headers: headers.map(([name, value]) => ({ name, value })) }) as EchoInfo
    expect(proxyFindings(echo([['X-Nettools-Probe', 'm'], ['X-Forwarded-For', '10.0.0.5']]), 'm', 'UA')).toEqual([{ id: 'clean', detail: '' }])
    const f = proxyFindings(echo([['Via', '1.1 squid'], ['X-Forwarded-For', '10.0.0.5, 172.19.0.1']], 'Other'), 'm', 'UA').map(x => x.id)
    expect(f).toEqual(['marker-missing', 'ua-changed', 'proxy-header', 'extra-hops'])
  })
})
