import { describe, expect, it } from 'vitest'
import { fromMilliwatts, linkBudget, toMilliwatts } from './optical-power.service'

describe('power conversion', () => {
  it('converts between units', () => {
    expect(toMilliwatts(0, 'dBm')).toBe(1)
    expect(toMilliwatts(-30, 'dBm')).toBeCloseTo(0.001)
    expect(toMilliwatts(0, 'dBW')).toBe(1000)
    expect(toMilliwatts(500, 'uW')).toBe(0.5)
    expect(toMilliwatts(0, 'mW')).toBeUndefined()
    const all = fromMilliwatts(2)
    expect(all.dBm).toBeCloseTo(3.0103, 3)
    expect(all.uW).toBe(2000)
    expect(all.dBW).toBeCloseTo(-26.9897, 3)
  })
})

describe('linkBudget', () => {
  const base = { txDbm: -3, rxSensitivityDbm: -20, lengthKm: 10, attenuationDbPerKm: 0.35, connectors: 2, connectorLossDb: 0.5, splices: 4, spliceLossDb: 0.1, marginDb: 3 }
  it('computes losses and headroom', () => {
    expect(linkBudget(base)).toEqual({ fiberLoss: 3.5, connectorLoss: 1, spliceLoss: 0.4, totalLoss: 4.9, rxDbm: -7.9, budget: 17, headroom: 9.1, overload: false, attenuatorDb: 0 })
  })
  it('flags receiver overload on short links', () => {
    const r = linkBudget({ ...base, txDbm: 5, lengthKm: 0.1, splices: 0, rxOverloadDbm: -1 })!
    expect(r.overload).toBe(true)
    expect(r.attenuatorDb).toBe(5)
  })
  it('rejects bad input', () => {
    expect(linkBudget({ ...base, lengthKm: -1 })).toBeUndefined()
    expect(linkBudget({ ...base, txDbm: Number.NaN })).toBeUndefined()
  })
})
