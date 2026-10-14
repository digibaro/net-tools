import { describe, expect, it } from 'vitest'
import { headerRows, proxyHints } from './request-headers.service'

const hdr = (pairs: Record<string, string>) => ({ headers: Object.entries(pairs).map(([name, value]) => ({ name, value })) })

describe('proxyHints', () => {
  it('reports nothing for a direct browser request through our nginx', () => {
    expect(proxyHints(hdr({ 'User-Agent': 'x', 'Accept-Encoding': 'gzip', 'X-Forwarded-For': '203.0.113.5', 'X-Real-Ip': '203.0.113.5' }))).toEqual([])
  })

  it('detects proxies, CDNs and stripped compression', () => {
    expect(proxyHints(hdr({
      'User-Agent': 'x',
      'Via': '1.1 squid',
      'Forwarded': 'for=198.51.100.7',
      'X-Forwarded-For': '198.51.100.7, 10.0.0.2',
      'Cf-Connecting-Ip': '198.51.100.7',
    }))).toEqual(['via', 'forwarded', 'multi-hop', 'cdn', 'compression-stripped'])
  })
})

describe('headerRows', () => {
  it('marks headers added by NetTools', () => {
    expect(headerRows(hdr({ 'Accept': '*/*', 'X-Real-Ip': '1.2.3.4' }))).toEqual([
      { name: 'Accept', value: '*/*', addedByNetTools: false },
      { name: 'X-Real-Ip', value: '1.2.3.4', addedByNetTools: true },
    ])
  })
})
