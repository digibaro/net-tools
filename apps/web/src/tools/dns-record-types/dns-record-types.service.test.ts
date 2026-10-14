import { describe, expect, it } from 'vitest'
import { COMMON_TYPES } from './dns-record-types.examples'
import { filterTypes, RR_TYPES } from './dns-record-types.service'

describe('dns record types', () => {
  it('every curated type exists in the IANA data', () => {
    const types = new Set(RR_TYPES.map(r => r.type))
    for (const t of Object.keys(COMMON_TYPES))
      expect(types.has(t), t).toBe(true)
  })

  it('filters by name, value and common flag', () => {
    expect(filterTypes('caa', true)[0]).toMatchObject({ value: 257, rfc: ['RFC 8659'] })
    expect(filterTypes('28', false)[0]?.type).toBe('AAAA')
    expect(filterTypes('', true).length).toBe(Object.keys(COMMON_TYPES).length)
    expect(filterTypes('', false).length).toBe(RR_TYPES.length)
  })
})
