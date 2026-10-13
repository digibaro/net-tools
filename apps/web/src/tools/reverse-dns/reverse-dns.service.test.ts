import { describe, expect, it } from 'vitest'
import { isIpAddress, verdict } from './reverse-dns.service'

const n = (forwardConfirmed: boolean) => ({ name: 'x.', forwardAddresses: [], forwardConfirmed })

describe('reverse-dns', () => {
  it('validates addresses', () => {
    expect(isIpAddress('192.0.2.1')).toBe(true)
    expect(isIpAddress('2001:db8::1')).toBe(true)
    expect(isIpAddress('example.com')).toBe(false)
  })
  it('summarizes forward confirmation', () => {
    expect(verdict({ names: [] })).toBe('none')
    expect(verdict({ names: [n(true), n(true)] })).toBe('confirmed')
    expect(verdict({ names: [n(true), n(false)] })).toBe('partial')
    expect(verdict({ names: [n(false)] })).toBe('unconfirmed')
  })
})
