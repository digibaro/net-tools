import { describe, expect, it } from 'vitest'
import { isBroadcast, normalizeMac } from './wake-on-lan.service'

describe('wake-on-lan', () => {
  it('normalizes MACs', () => {
    expect(normalizeMac('00-11-22-AA-BB-CC')).toBe('00:11:22:aa:bb:cc')
    expect(normalizeMac('0011.22aa.bbcc')).toBe('00:11:22:aa:bb:cc')
    expect(normalizeMac('00:11:22')).toBeUndefined()
  })
  it('accepts private broadcasts only', () => {
    expect(isBroadcast('')).toBe(true)
    expect(isBroadcast('192.168.1.255')).toBe(true)
    expect(isBroadcast('8.8.8.255')).toBe(false)
  })
})
