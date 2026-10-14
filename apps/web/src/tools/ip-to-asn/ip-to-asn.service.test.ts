import { describe, expect, it } from 'vitest'
import { rirName } from './ip-to-asn.service'

describe('rirName', () => {
  it('names registries', () => {
    expect(rirName('ripencc')).toBe('RIPE NCC')
    expect(rirName('xyz')).toBe('XYZ')
  })
})
