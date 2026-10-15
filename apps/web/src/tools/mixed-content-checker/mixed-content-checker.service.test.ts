import type { MixedResult } from './mixed-content-checker.service'
import { describe, expect, it } from 'vitest'
import { verdict } from './mixed-content-checker.service'

const r = (active: number, passive: number, forms: number, upgrade = false) => ({ active, passive, forms, links: 3, upgradeInsecureRequests: upgrade }) as MixedResult

describe('mixed-content-checker', () => {
  it('rates the page', () => {
    expect(verdict(r(0, 0, 0))).toBe('clean')
    expect(verdict(r(0, 2, 0))).toBe('passive')
    expect(verdict(r(1, 0, 0))).toBe('active')
    expect(verdict(r(1, 0, 0, true))).toBe('upgraded')
  })
})
