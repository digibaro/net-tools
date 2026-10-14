import type { Unit } from '@/utils/units'
import { formatNumber, RATE_UNITS, SIZE_UNITS } from '@/utils/units'

export type Quantity = 'size' | 'rate'

export interface ConversionRow {
  unit: string
  value: string
  /** Raw number for JSON export. */
  raw: number
}

export function unitsFor(q: Quantity): Unit[] {
  return q === 'size' ? SIZE_UNITS : RATE_UNITS
}

export function convertAll(value: number, unitId: string, q: Quantity): ConversionRow[] | undefined {
  const units = unitsFor(q)
  const from = units.find(u => u.id === unitId)
  if (!from || !Number.isFinite(value))
    return undefined
  const bits = value * from.factor
  return units.map(u => ({ unit: u.label, value: formatNumber(bits / u.factor, 8), raw: bits / u.factor }))
}
