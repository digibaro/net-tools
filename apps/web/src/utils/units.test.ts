import { describe, expect, it } from 'vitest'
import { formatDuration, formatNumber, humanize, RATE_UNITS, SIZE_UNITS, unitFactor } from './units'

describe('units', () => {
  it('has the expected factors', () => {
    expect(unitFactor(SIZE_UNITS, 'MB')).toBe(8e6)
    expect(unitFactor(SIZE_UNITS, 'MiB')).toBe(8 * 2 ** 20)
    expect(unitFactor(SIZE_UNITS, 'Gb')).toBe(1e9)
    expect(unitFactor(RATE_UNITS, 'Mbps')).toBe(1e6)
    expect(unitFactor(RATE_UNITS, 'MBps')).toBe(8e6)
    expect(unitFactor(RATE_UNITS, 'nope')).toBeUndefined()
    expect(new Set(SIZE_UNITS.map(u => u.id)).size).toBe(SIZE_UNITS.length)
  })

  it.each([
    [0, '0'],
    [1234567.891, '1,234,570'],
    [0.000123456, '0.000123456'],
    [1e-9, '1.00000e-9'],
    [Infinity, '∞'],
  ])('formatNumber(%d) = %s', (n, s) => {
    expect(formatNumber(n)).toBe(s)
  })

  it.each([
    [0.0000025, '2.5 µs'],
    [0.25, '250 ms'],
    [42, '42 s'],
    [3725.5, '1 h 2 min 5.5 s'],
    [90061, '1 d 1 h 1 min 1 s'],
    [0, '0 s'],
  ])('formatDuration(%d) = %s', (s, want) => {
    expect(formatDuration(s)).toBe(want)
  })

  it('humanize', () => {
    expect(humanize(8e9, 'size-bytes')).toBe('1 GB')
    expect(humanize(1.5e6, 'rate-bits')).toBe('1.5 Mbit/s')
    expect(humanize(999, 'size-bits')).toBe('999 bit')
  })
})
