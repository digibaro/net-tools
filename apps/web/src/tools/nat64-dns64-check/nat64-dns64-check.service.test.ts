import type { Dns64Result } from './nat64-dns64-check.service'
import { describe, expect, it } from 'vitest'
import { isResolver, verdict } from './nat64-dns64-check.service'

describe('nat64-dns64-check', () => {
  it('validates the resolver field', () => {
    expect(isResolver('')).toBe(true)
    expect(isResolver('9.9.9.9')).toBe(true)
    expect(isResolver('2001:4860:4860::6464')).toBe(true)
    expect(isResolver('[2001:db8::53]:5353')).toBe(true)
    expect(isResolver('dns.example')).toBe(false)
  })
  it('rates the result', () => {
    expect(verdict({ dns64: false, prefixes: [] } as unknown as Dns64Result)).toBe('none')
    expect(verdict({ dns64: true, prefixes: [{ prefix: '64:ff9b::/96', length: 96, wellKnown: true }] } as Dns64Result)).toBe('wellKnown')
    expect(verdict({ dns64: true, prefixes: [{ prefix: '2001:db8:64::/96', length: 96, wellKnown: false }] } as Dns64Result)).toBe('networkSpecific')
  })
})
