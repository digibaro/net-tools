import type { ChainResult } from './cert-chain-validator.service'
import { describe, expect, it } from 'vitest'
import { countCerts, verdict } from './cert-chain-validator.service'

describe('cert-chain-validator', () => {
  it('rates the chain', () => {
    const r = (trusted: boolean, complete: boolean, ordered: boolean) => ({ trusted, complete, ordered }) as ChainResult
    expect(verdict(r(true, true, true))).toBe('ok')
    expect(verdict(r(true, true, false))).toBe('unordered')
    expect(verdict(r(true, false, true))).toBe('incomplete')
    expect(verdict(r(false, false, true))).toBe('untrusted')
  })
  it('counts certificates', () => {
    expect(countCerts('-----BEGIN CERTIFICATE-----\nx\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----')).toBe(2)
  })
})
