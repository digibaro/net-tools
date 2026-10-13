import { describe, expect, it, vi } from 'vitest'
import { fetchPublicIp, IPIFY, relation } from './what-is-my-ip.service'

const respond = (body: unknown, status = 200): typeof fetch => vi.fn(async () => new Response(JSON.stringify(body), { status }))

describe('fetchPublicIp', () => {
  it('returns a validated address of the requested family', async () => {
    const f = respond({ ip: '203.0.113.7' })
    await expect(fetchPublicIp(4, f)).resolves.toBe('203.0.113.7')
    expect(f).toHaveBeenCalledWith(IPIFY.v4, expect.objectContaining({ referrerPolicy: 'no-referrer' }))
    await expect(fetchPublicIp(6, respond({ ip: '2001:DB8::1' }))).resolves.toBe('2001:db8::1')
  })

  it.each([
    ['wrong family', respond({ ip: '203.0.113.7' }), 6],
    ['not an IP', respond({ ip: '<script>' }), 4],
    ['bad shape', respond(['203.0.113.7']), 4],
    ['HTTP error', respond({ ip: '203.0.113.7' }, 500), 4],
    ['network error', vi.fn(async () => { throw new TypeError('offline') }) as typeof fetch, 4],
  ] as const)('returns undefined on %s', async (_, f, family) => {
    await expect(fetchPublicIp(family, f)).resolves.toBeUndefined()
  })
})

describe('relation', () => {
  it('compares the server view with the public view', () => {
    expect(relation('203.0.113.7', ['203.0.113.7', undefined])).toBe('same')
    expect(relation('192.168.1.20', ['203.0.113.7'])).toBe('private')
    expect(relation('8.8.4.4', ['203.0.113.7'])).toBe('different')
  })
})
