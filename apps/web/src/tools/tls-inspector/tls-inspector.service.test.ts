import { describe, expect, it } from 'vitest'
import { expiryType, isLegacyVersion } from './tls-inspector.service'

describe('tls-inspector', () => {
  it('flags legacy TLS', () => {
    expect(isLegacyVersion('TLS 1.1')).toBe(true)
    expect(isLegacyVersion('TLS 1.3')).toBe(false)
  })
  it('rates expiry', () => {
    expect(expiryType(-1)).toBe('error')
    expect(expiryType(10)).toBe('warning')
    expect(expiryType(90)).toBe('success')
  })
})
