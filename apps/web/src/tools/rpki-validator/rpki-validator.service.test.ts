import { describe, expect, it } from 'vitest'
import { statusType } from './rpki-validator.service'

describe('statusType', () => {
  it('maps RPKI states', () => {
    expect(statusType('valid')).toBe('success')
    expect(statusType('invalid_asn')).toBe('error')
    expect(statusType('unknown')).toBe('default')
  })
})
