import { describe, expect, it } from 'vitest'
import { formatBytes, formatRate } from './iperf.service'

describe('iperf', () => {
  it.each([[12409.5, '12.4 Gbit/s'], [1500, '1.50 Gbit/s'], [940.2, '940 Mbit/s'], [9.99, '10.0 Mbit/s'], [0.35, '350 kbit/s']])('formatRate(%s)', (v, want) => {
    expect(formatRate(v)).toBe(want)
  })
  it('formats bytes', () => {
    expect(formatBytes(3109027840)).toBe('2.9 GiB')
    expect(formatBytes(512)).toBe('512 B')
  })
})
