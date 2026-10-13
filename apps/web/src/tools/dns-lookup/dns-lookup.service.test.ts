import { describe, expect, it, vi } from 'vitest'
import { fromDohJson, isValidQueryName, lookupInBrowser } from './dns-lookup.service'

describe('isValidQueryName', () => {
  it.each(['example.com', 'münchen.de', '_dmarc.example.com', '192.0.2.1', '2001:db8::1'])('accepts %s', (s) => {
    expect(isValidQueryName(s)).toBe(true)
  })
  it.each(['', 'exa mple.com', 'http://x.com'])('rejects %j', (s) => {
    expect(isValidQueryName(s)).toBe(false)
  })
})

describe('fromDohJson', () => {
  it('maps the DoH JSON format', () => {
    const r = fromDohJson({ Status: 0, RA: true, AD: true, Answer: [{ name: 'example.com', type: 15, TTL: 300, data: '10 mail.example.com.' }] }, 'example.com', 'MX', 'google', 12.3456)
    expect(r).toMatchObject({ rcode: 'NOERROR', transport: 'doh', rttMs: 12.346, flags: { ra: true, ad: true, tc: false }, answer: [{ name: 'example.com.', type: 'MX', ttl: 300, data: '10 mail.example.com.' }] })
    expect(fromDohJson({ Status: 3 }, 'x', 'A', 'cloudflare', 1).rcode).toBe('NXDOMAIN')
  })
})

describe('lookupInBrowser', () => {
  it('queries the provider with the dns-json accept header', async () => {
    const f = vi.fn(async () => new Response(JSON.stringify({ Status: 0, Answer: [] })))
    const r = await lookupInBrowser('münchen.de', 'A', 'cloudflare', f as unknown as typeof fetch)
    expect(f).toHaveBeenCalledWith('https://cloudflare-dns.com/dns-query?name=xn--mnchen-3ya.de&type=A', expect.objectContaining({ headers: { Accept: 'application/dns-json' } }))
    expect(r.name).toBe('xn--mnchen-3ya.de')
  })

  it('throws on HTTP errors', async () => {
    const f = vi.fn(async () => new Response('x', { status: 400 }))
    await expect(lookupInBrowser('example.com', 'A', 'google', f as unknown as typeof fetch)).rejects.toThrow('HTTP 400')
  })
})
