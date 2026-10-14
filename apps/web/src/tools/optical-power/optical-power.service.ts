export type PowerUnit = 'dBm' | 'mW' | 'uW' | 'W' | 'dBW'

export const POWER_UNITS: PowerUnit[] = ['dBm', 'mW', 'uW', 'W', 'dBW']

/** Converts a power value to milliwatts; undefined for impossible inputs (≤ 0 linear power). */
export function toMilliwatts(value: number, unit: PowerUnit): number | undefined {
  if (!Number.isFinite(value))
    return undefined
  switch (unit) {
    case 'dBm':
      return 10 ** (value / 10)
    case 'dBW':
      return 10 ** (value / 10) * 1000
    case 'mW':
      return value > 0 ? value : undefined
    case 'uW':
      return value > 0 ? value / 1000 : undefined
    case 'W':
      return value > 0 ? value * 1000 : undefined
  }
}

export function fromMilliwatts(mw: number): Record<PowerUnit, number> {
  const dBm = 10 * Math.log10(mw)
  return { dBm, mW: mw, uW: mw * 1000, W: mw / 1000, dBW: dBm - 30 }
}

/** Typical fibre attenuation in dB/km (ITU-T G.652 single-mode, OM3/OM4 multimode). */
export const FIBER_PRESETS = [
  { id: 'smf1310', attenuation: 0.35 },
  { id: 'smf1550', attenuation: 0.22 },
  { id: 'mmf850', attenuation: 3.0 },
  { id: 'mmf1300', attenuation: 1.0 },
] as const

export interface Budget {
  txDbm: number
  rxSensitivityDbm: number
  /** Receiver overload (maximum input power); optional. */
  rxOverloadDbm?: number
  lengthKm: number
  attenuationDbPerKm: number
  connectors: number
  connectorLossDb: number
  splices: number
  spliceLossDb: number
  /** Safety margin for ageing, repairs and temperature. */
  marginDb: number
}

export interface BudgetResult {
  fiberLoss: number
  connectorLoss: number
  spliceLoss: number
  totalLoss: number
  /** Expected power at the receiver. */
  rxDbm: number
  /** Power budget: Tx − sensitivity. */
  budget: number
  /** Budget left after losses and the safety margin; negative means the link will not work reliably. */
  headroom: number
  overload: boolean
  /** Attenuation still needed to stay below the receiver overload level. */
  attenuatorDb: number
}

const round = (n: number) => Math.round(n * 100) / 100

export function linkBudget(b: Budget): BudgetResult | undefined {
  const values = [b.txDbm, b.rxSensitivityDbm, b.lengthKm, b.attenuationDbPerKm, b.connectors, b.connectorLossDb, b.splices, b.spliceLossDb, b.marginDb]
  if (values.some(v => !Number.isFinite(v)) || b.lengthKm < 0 || b.connectors < 0 || b.splices < 0)
    return undefined
  const fiberLoss = b.lengthKm * b.attenuationDbPerKm
  const connectorLoss = b.connectors * b.connectorLossDb
  const spliceLoss = b.splices * b.spliceLossDb
  const totalLoss = fiberLoss + connectorLoss + spliceLoss
  const rxDbm = b.txDbm - totalLoss
  const budget = b.txDbm - b.rxSensitivityDbm
  const overload = b.rxOverloadDbm !== undefined && Number.isFinite(b.rxOverloadDbm) && rxDbm > b.rxOverloadDbm
  return {
    fiberLoss: round(fiberLoss),
    connectorLoss: round(connectorLoss),
    spliceLoss: round(spliceLoss),
    totalLoss: round(totalLoss),
    rxDbm: round(rxDbm),
    budget: round(budget),
    headroom: round(budget - totalLoss - b.marginDb),
    overload,
    attenuatorDb: overload ? Math.ceil(rxDbm - b.rxOverloadDbm!) : 0,
  }
}
