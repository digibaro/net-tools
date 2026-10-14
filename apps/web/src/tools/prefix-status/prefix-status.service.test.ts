import type { PrefixStatus } from './prefix-status.service'
import { describe, expect, it } from 'vitest'
import { visibilityPct } from './prefix-status.service'

describe('visibilityPct', () => {
  it('computes visibility', () => {
    expect(visibilityPct({ seenBy: 348, totalPeers: 352 } as PrefixStatus)).toBe(98.9)
    expect(visibilityPct({ seenBy: 0, totalPeers: 0 } as PrefixStatus)).toBe(0)
  })
})
