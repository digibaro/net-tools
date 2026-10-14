import { describe, expect, it } from 'vitest'
import { keyLine } from './dnssec-validator.service'

describe('keyLine', () => {
  it('formats keys', () => {
    expect(keyLine({ tag: 38696, flags: 257, role: 'KSK', algorithm: 'RSASHA256', trusted: true })).toBe('KSK 38696 RSASHA256 ✓')
  })
})
