import { describe, expect, it } from 'vitest'
import { portSpeed } from './peeringdb-lookup.service'

describe('portSpeed', () => {
  it.each([[100000, '100G'], [2500, '2.5G'], [1000, '1G'], [500, '500M']])('%d → %s', (n, want) => {
    expect(portSpeed(n)).toBe(want)
  })
})
