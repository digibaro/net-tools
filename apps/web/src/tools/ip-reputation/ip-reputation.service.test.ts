import { describe, expect, it } from 'vitest'
import { scoreLevel } from './ip-reputation.service'

describe('scoreLevel', () => {
  it.each([[0, 'clean'], [24, 'clean'], [25, 'suspicious'], [87, 'malicious']] as const)('%d → %s', (s, want) => {
    expect(scoreLevel(s)).toBe(want)
  })
})
