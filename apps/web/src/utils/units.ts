import { numberLocale } from './format'
/** Data size and data rate units, expressed in bits (sizes) or bits per second (rates). */
export interface Unit {
  id: string
  label: string
  factor: number
}

const dec = [['k', 1e3], ['M', 1e6], ['G', 1e9], ['T', 1e12], ['P', 1e15]] as const
const bin = [['Ki', 2 ** 10], ['Mi', 2 ** 20], ['Gi', 2 ** 30], ['Ti', 2 ** 40], ['Pi', 2 ** 50]] as const

export const SIZE_UNITS: Unit[] = [
  { id: 'b', label: 'bit', factor: 1 },
  ...dec.map(([p, f]) => ({ id: `${p}b`, label: `${p}bit`, factor: f })),
  ...bin.map(([p, f]) => ({ id: `${p}b`, label: `${p}bit`, factor: f })),
  { id: 'B', label: 'B (byte)', factor: 8 },
  ...dec.map(([p, f]) => ({ id: `${p}B`, label: `${p}B`, factor: 8 * f })),
  ...bin.map(([p, f]) => ({ id: `${p}B`, label: `${p}B`, factor: 8 * f })),
]

export const RATE_UNITS: Unit[] = [
  { id: 'bps', label: 'bit/s', factor: 1 },
  ...dec.slice(0, 4).map(([p, f]) => ({ id: `${p}bps`, label: `${p}bit/s`, factor: f })),
  { id: 'Bps', label: 'B/s', factor: 8 },
  ...dec.slice(0, 4).map(([p, f]) => ({ id: `${p}Bps`, label: `${p}B/s`, factor: 8 * f })),
  ...bin.slice(0, 3).map(([p, f]) => ({ id: `${p}Bps`, label: `${p}B/s`, factor: 8 * f })),
]

export function unitFactor(units: Unit[], id: string): number | undefined {
  return units.find(u => u.id === id)?.factor
}

/** Up to `digits` significant digits, no exponent for everyday magnitudes, thousands separators. */
export function formatNumber(n: number, digits = 6): string {
  if (!Number.isFinite(n))
    return '∞'
  if (n === 0)
    return '0'
  const abs = Math.abs(n)
  if (abs >= 1e21 || abs < 1e-6)
    return n.toExponential(digits - 1)
  return Number(n.toPrecision(digits)).toLocaleString(numberLocale(), { maximumFractionDigits: 20 })
}

/** "1 d 2 h 3 min 4.5 s"; sub-second values in ms / µs. */
export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds))
    return '∞'
  if (seconds === 0)
    return '0 s'
  if (seconds < 1e-3)
    return `${formatNumber(seconds * 1e6, 4)} µs`
  if (seconds < 1)
    return `${formatNumber(seconds * 1e3, 4)} ms`
  const parts: string[] = []
  let rest = seconds
  for (const [label, size] of [['y', 31_557_600], ['d', 86_400], ['h', 3600], ['min', 60]] as const) {
    if (rest >= size) {
      parts.push(`${Math.floor(rest / size)} ${label}`)
      rest %= size
    }
  }
  if (rest > 0 || !parts.length)
    parts.push(`${formatNumber(rest, parts.length ? 3 : 4)} s`)
  return parts.slice(0, 4).join(' ')
}

/** Picks the most readable unit (decimal, bits or bytes) for a value in bits. */
export function humanize(bits: number, kind: 'size-bytes' | 'size-bits' | 'rate-bits' | 'rate-bytes'): string {
  const byBytes = kind.endsWith('bytes')
  const v = byBytes ? bits / 8 : bits
  const suffixes = kind.startsWith('rate') ? (byBytes ? ['B/s', 'kB/s', 'MB/s', 'GB/s', 'TB/s'] : ['bit/s', 'kbit/s', 'Mbit/s', 'Gbit/s', 'Tbit/s']) : (byBytes ? ['B', 'kB', 'MB', 'GB', 'TB', 'PB'] : ['bit', 'kbit', 'Mbit', 'Gbit', 'Tbit', 'Pbit'])
  let i = 0
  let x = v
  while (Math.abs(x) >= 1000 && i < suffixes.length - 1) {
    x /= 1000
    i++
  }
  return `${formatNumber(x, 4)} ${suffixes[i]}`
}
