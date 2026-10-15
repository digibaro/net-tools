/** Seconds per period (month = 1/12 of a 365.25-day year, as most SLAs use). */
export const PERIODS = { day: 86400, week: 604800, month: 2629800, quarter: 7889400, year: 31557600 } as const
export type Period = keyof typeof PERIODS

/** Allowed downtime in seconds for an availability percentage. */
export const downtime = (pct: number, period: Period) => (1 - pct / 100) * PERIODS[period]

/** Availability percentage for a downtime (seconds) per period. */
export const availability = (seconds: number, period: Period) => (1 - seconds / PERIODS[period]) * 100

/** Parses "99.95", "99,95 %" or "4 nines" (= 99.99). */
export function parsePercent(s: string): number | undefined {
  const v = s.trim().replace(',', '.').replace(/%$/, '').trim()
  const nines = /^(\d+(?:\.\d+)?)\s*nines?$/i.exec(v)
  const n = nines ? 100 - 100 * 10 ** -Number(nines[1]) : Number(v)
  return v !== '' && Number.isFinite(n) && n >= 0 && n <= 100 ? n : undefined
}

/** "1 d 2 h", "52 min 35 s", "8.6 s", "864 ms". */
export function human(seconds: number): string {
  if (seconds < 1)
    return `${Math.round(seconds * 1000)} ms`
  if (seconds < 60)
    return `${Math.round(seconds * 10) / 10} s`
  const parts: string[] = []
  let s = Math.round(seconds)
  for (const [u, n] of [['d', 86400], ['h', 3600], ['min', 60], ['s', 1]] as const) {
    if (s >= n && parts.length < 2) {
      parts.push(`${Math.floor(s / n)} ${u}`)
      s %= n
    }
    else if (parts.length) {
      break
    }
  }
  return parts.join(' ')
}

/** Components that all must work (in series) multiply; redundant ones (in parallel) fail only together. */
export const series = (pcts: number[]) => pcts.reduce((a, p) => a * p / 100, 1) * 100
export const parallel = (pcts: number[]) => (1 - pcts.reduce((a, p) => a * (1 - p / 100), 1)) * 100

/** Number of nines: 99.95 → 3.3 */
export const nines = (pct: number) => (pct >= 100 ? Infinity : -Math.log10(1 - pct / 100))
