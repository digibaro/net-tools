import type { DaneResult } from './dane-checker.service'
import { describe, expect, it } from 'vitest'
import { describeRecord, overall } from './dane-checker.service'

describe('dane-checker', () => {
  it('describes TLSA parameters', () => {
    expect(describeRecord({ usage: 3, selector: 1, matchingType: 1, data: '', match: true, matchedCert: 0 })).toBe('3 1 1 (DANE-EE SPKI SHA2-256)')
  })
  it('takes the worst host verdict', () => {
    const r = (...v: string[]) => ({ hosts: v.map(verdict => ({ verdict })) }) as DaneResult
    expect(overall(r('valid', 'valid'))).toBe('valid')
    expect(overall(r('valid', 'mismatch'))).toBe('mismatch')
    expect(overall(r('no-tlsa'))).toBe('no-tlsa')
  })
})
