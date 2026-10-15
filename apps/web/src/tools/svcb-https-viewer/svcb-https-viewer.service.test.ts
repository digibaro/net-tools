import type { SvcbRecord } from './svcb-https-viewer.service'
import { describe, expect, it } from 'vitest'
import { isOwnerName, offers } from './svcb-https-viewer.service'

describe('svcb-https-viewer', () => {
  it('accepts underscore owner names', () => {
    expect(isOwnerName('_dns.resolver.arpa')).toBe(true)
    expect(isOwnerName('cloudflare.com')).toBe(true)
    expect(isOwnerName('bad name')).toBe(false)
  })
  it('summarises a service record', () => {
    const r = { params: [{ key: 'alpn', value: 'h3,h2' }, { key: 'ipv4hint', value: '192.0.2.1' }], ech: [{}] } as unknown as SvcbRecord
    expect(offers(r)).toEqual(['h3', 'h2', 'ECH', 'hints'])
  })
})
