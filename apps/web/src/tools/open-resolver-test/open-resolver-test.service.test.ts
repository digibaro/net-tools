import type { OpenResolverResult } from './open-resolver-test.service'
import { describe, expect, it } from 'vitest'
import { isServer, verdict } from './open-resolver-test.service'

describe('open-resolver-test', () => {
  it('rates the answer', () => {
    expect(verdict({ responds: false, open: false } as OpenResolverResult)).toBe('silent')
    expect(verdict({ responds: true, open: true } as OpenResolverResult)).toBe('open')
    expect(verdict({ responds: true, open: false } as OpenResolverResult)).toBe('closed')
  })
  it('validates the server', () => {
    expect(isServer('203.0.113.53')).toBe(true)
    expect(isServer('ns1.example.com:5353')).toBe(true)
    expect(isServer('2001:db8::53')).toBe(true)
    expect(isServer('a b')).toBe(false)
  })
})
