export const MAX_UNSCALED_WINDOW = 65_535
export const MAX_WINDOW_SCALE = 14 // RFC 7323

export interface BdpResult {
  bdpBits: number
  bdpBytes: number
  /** Window scale shift needed so 65535 << shift >= BDP; null if beyond RFC 7323's maximum. */
  windowScale: number | null
}

export function bdp(rateBps: number, rttMs: number): BdpResult | undefined {
  if (!(rateBps > 0) || !(rttMs > 0))
    return undefined
  const bdpBits = rateBps * rttMs / 1000
  const bdpBytes = bdpBits / 8
  let shift = 0
  while ((MAX_UNSCALED_WINDOW * 2 ** shift) < bdpBytes && shift <= MAX_WINDOW_SCALE)
    shift++
  return { bdpBits, bdpBytes, windowScale: shift > MAX_WINDOW_SCALE ? null : shift }
}

/** Max TCP throughput (bit/s) with a given receive window and RTT. */
export function windowThroughput(windowBytes: number, rttMs: number): number | undefined {
  if (!(windowBytes > 0) || !(rttMs > 0))
    return undefined
  return windowBytes * 8 / (rttMs / 1000)
}

/** Mathis et al. (1997): throughput ≤ (MSS / RTT) · (C / √p), C ≈ 1.22. */
export function mathisThroughput(mssBytes: number, rttMs: number, lossPercent: number): number | undefined {
  const p = lossPercent / 100
  if (!(mssBytes > 0) || !(rttMs > 0) || !(p > 0) || p >= 1)
    return undefined
  return (mssBytes * 8 / (rttMs / 1000)) * (1.22 / Math.sqrt(p))
}
