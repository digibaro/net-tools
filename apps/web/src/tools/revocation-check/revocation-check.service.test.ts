import type { Revocation } from './revocation-check.service'
import { describe, expect, it } from 'vitest'
import { crlOnly } from './revocation-check.service'

describe('crlOnly', () => {
  it('detects CRL-only certificates', () => {
    expect(crlOnly({ ocsp: [], crl: [{}] } as unknown as Revocation)).toBe(true)
    expect(crlOnly({ ocsp: [{ source: 'http://ocsp.example' }], crl: [{}] } as unknown as Revocation)).toBe(false)
  })
})
