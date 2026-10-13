import { describe, expect, it } from 'vitest'
import { offsetQuality } from './ntp-check.service'

describe('offsetQuality', () => {
  it('rates offsets', () => {
    expect(offsetQuality(-3)).toBe('good')
    expect(offsetQuality(120)).toBe('fair')
    expect(offsetQuality(-2000)).toBe('poor')
  })
})
