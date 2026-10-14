import { describe, expect, it } from 'vitest'
import { isScannable } from './arp-scan.service'

describe('isScannable', () => {
  it.each([['192.168.1.0/24', true], ['10.0.0.0/22', true], ['10.0.0.0/16', false], ['192.168.1.1', false], ['fd00::/120', false]])('%s → %s', (s, want) => {
    expect(isScannable(s)).toBe(want)
  })
})
