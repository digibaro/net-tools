import type { RoaCoverage } from './roa-coverage.service'
import { describe, expect, it } from 'vitest'
import { coverage, suggestions } from './roa-coverage.service'

describe('roa-coverage', () => {
  it('computes coverage and suggestions', () => {
    const r = { asn: 64500, valid: 3, invalid: 1, notFound: 1, unchecked: 2, prefixes: [{ prefix: '192.0.2.0/24', status: 'not-found', reason: '', roa: '' }] } as RoaCoverage
    expect(coverage(r)).toBe(60)
    expect(suggestions(r)).toEqual(['192.0.2.0/24 maxLength 24 AS64500'])
  })
})
