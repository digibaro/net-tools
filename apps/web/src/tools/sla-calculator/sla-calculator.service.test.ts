import { describe, expect, it } from 'vitest'
import { availability, downtime, human, nines, parallel, parsePercent, series } from './sla-calculator.service'

describe('sla-calculator', () => {
  it('converts availability and downtime', () => {
    expect(downtime(99.9, 'year') / 3600).toBeCloseTo(8.766, 3)
    expect(human(downtime(99.99, 'month'))).toBe('4 min 23 s')
    expect(availability(3600, 'day')).toBeCloseTo(95.833, 3)
    expect(human(0.8)).toBe('800 ms')
  })
  it('parses input', () => {
    expect(parsePercent('99,95 %')).toBe(99.95)
    expect(parsePercent('4 nines')).toBeCloseTo(99.99, 9)
    expect(parsePercent('101')).toBeUndefined()
  })
  it('combines components', () => {
    expect(series([99.9, 99.9])).toBeCloseTo(99.8001, 4)
    expect(parallel([99, 99])).toBeCloseTo(99.99, 6)
    expect(nines(99.95)).toBeCloseTo(3.301, 3)
  })
})
