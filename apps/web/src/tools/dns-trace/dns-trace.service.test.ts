import { describe, expect, it } from 'vitest'
import { traceText } from './dns-trace.service'

describe('traceText', () => {
  it('renders referrals and answers', () => {
    const txt = traceText([
      { step: 1, zone: '.', server: 'a.root-servers.net.', address: '198.41.0.4', rcode: 'NOERROR', aa: false, rttMs: 10, answer: [], referral: ['a.gtld-servers.net.'], note: '' },
      { step: 2, zone: 'com.', server: 'ns.example.', address: '192.0.2.1', rcode: 'NOERROR', aa: true, rttMs: 5, answer: [{ name: 'example.com.', ttl: 60, class: 'IN', type: 'A', data: '192.0.2.10' }], referral: [], note: '' },
    ])
    expect(txt).toBe(';; . via a.root-servers.net. (198.41.0.4) NOERROR in 10 ms\n  → a.gtld-servers.net.\n\n;; com. via ns.example. (192.0.2.1) NOERROR AA in 5 ms\nexample.com.\t60\tIN\tA\t192.0.2.10')
  })
})
