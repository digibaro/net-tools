import { describe, expect, it } from 'vitest'
import { convertAll } from './data-unit-converter.service'

describe('convertAll', () => {
  it('converts sizes between bits/bytes and decimal/binary prefixes', () => {
    const rows = convertAll(1, 'GiB', 'size')!
    const v = (u: string) => rows.find(r => r.unit === u)!.raw
    expect(v('B (byte)')).toBe(1073741824)
    expect(v('MiB')).toBe(1024)
    expect(v('MB')).toBeCloseTo(1073.741824)
    expect(v('Gbit')).toBeCloseTo(8.589934592)
  })

  it('converts rates', () => {
    const rows = convertAll(100, 'Mbps', 'rate')!
    expect(rows.find(r => r.unit === 'MB/s')!.value).toBe('12.5')
    expect(rows.find(r => r.unit === 'MiB/s')!.value).toBe('11.920929')
  })

  it('rejects unknown units and non-numbers', () => {
    expect(convertAll(1, 'nope', 'size')).toBeUndefined()
    expect(convertAll(Number.NaN, 'B', 'size')).toBeUndefined()
  })
})
