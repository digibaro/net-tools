import { describe, expect, it } from 'vitest'
import { isScanPrefix, status } from './ptr-coverage.service'

describe('ptr-coverage', () => {
  it('accepts at most 256 addresses', () => {
    expect(isScanPrefix('192.0.2.0/24')).toBe(true)
    expect(isScanPrefix('192.0.2.7')).toBe(true)
    expect(isScanPrefix('2001:db8::/120')).toBe(true)
    expect(isScanPrefix('192.0.0.0/23')).toBe(false)
    expect(isScanPrefix('2001:db8::/64')).toBe(false)
  })
  it('classifies rows', () => {
    expect(status({ ip: '', rcode: 'NOERROR', names: ['a'], confirmed: true, error: '' })).toBe('confirmed')
    expect(status({ ip: '', rcode: 'NOERROR', names: ['a'], confirmed: false, error: '' })).toBe('mismatch')
    expect(status({ ip: '', rcode: 'NXDOMAIN', names: [], confirmed: false, error: '' })).toBe('none')
  })
})
