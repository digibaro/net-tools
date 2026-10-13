/**
 * Protocol efficiency = payload / bytes on the wire per full-size frame (1500 MTU).
 * Ethernet on the wire: preamble+SFD 8 + header 14 + FCS 4 + inter-frame gap 12 = 38 bytes.
 */
export const OVERHEAD_PRESETS = {
  'none': 1,
  'tcp-ipv4': 1460 / 1538,
  'tcp-ipv6': 1440 / 1538,
  'udp-ipv4': 1472 / 1538,
} as const

export type OverheadPreset = keyof typeof OVERHEAD_PRESETS | 'custom'

export function efficiency(preset: OverheadPreset, customPercent = 0): number {
  if (preset === 'custom')
    return Math.min(Math.max(1 - customPercent / 100, 0.000001), 1)
  return OVERHEAD_PRESETS[preset]
}

/** Seconds to transfer `sizeBits` over a link of `rateBps` at the given efficiency. */
export function transferTime(sizeBits: number, rateBps: number, eff: number): number | undefined {
  if (!(sizeBits >= 0) || !(rateBps > 0) || !(eff > 0))
    return undefined
  return sizeBits / (rateBps * eff)
}

/** Link rate (bit/s) needed to move `sizeBits` in `seconds`. */
export function requiredRate(sizeBits: number, seconds: number, eff: number): number | undefined {
  if (!(sizeBits >= 0) || !(seconds > 0) || !(eff > 0))
    return undefined
  return sizeBits / seconds / eff
}

export const TIME_UNITS = { s: 1, min: 60, h: 3600, d: 86_400 } as const
export type TimeUnit = keyof typeof TIME_UNITS
