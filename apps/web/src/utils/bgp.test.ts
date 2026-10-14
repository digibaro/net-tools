import { describe, expect, it } from 'vitest'
import { isPrefixOrIp, parseAsn } from './bgp'

describe('bgp utils', () => {
  it('parses AS numbers', () => {
    expect(parseAsn('AS13335')).toBe(13335)
    expect(parseAsn('as 64500')).toBe(64500)
    expect(parseAsn('13335')).toBe(13335)
    expect(parseAsn('4294967296')).toBeUndefined()
    expect(parseAsn('ASX')).toBeUndefined()
  })
  it('accepts prefixes and addresses', () => {
    expect(isPrefixOrIp('1.1.1.0/24')).toBe(true)
    expect(isPrefixOrIp('2606:4700::/32')).toBe(true)
    expect(isPrefixOrIp('8.8.8.8')).toBe(true)
    expect(isPrefixOrIp('nope')).toBe(false)
  })
})
