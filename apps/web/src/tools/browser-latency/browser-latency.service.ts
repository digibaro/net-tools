export const MAX_SAMPLES = 100
export const MIN_INTERVAL = 250
export const REQUEST_TIMEOUT = 5000

export interface Stats {
  sent: number
  received: number
  lossPct: number
  min: number
  avg: number
  median: number
  p95: number
  max: number
  stddev: number
  /** Mean absolute difference between consecutive successful samples (RFC 3550 style, unsmoothed). */
  jitter: number
}

const r1 = (n: number) => Math.round(n * 10) / 10

function percentile(sorted: number[], p: number): number {
  const i = (sorted.length - 1) * p
  const lo = Math.floor(i)
  const hi = Math.ceil(i)
  return sorted[lo]! + (sorted[hi]! - sorted[lo]!) * (i - lo)
}

export function stats(samples: (number | null)[]): Stats | undefined {
  const ok = samples.filter((s): s is number => s !== null)
  if (!samples.length)
    return undefined
  const sent = samples.length
  if (!ok.length)
    return { sent, received: 0, lossPct: 100, min: 0, avg: 0, median: 0, p95: 0, max: 0, stddev: 0, jitter: 0 }
  const sorted = [...ok].sort((a, b) => a - b)
  const avg = ok.reduce((a, b) => a + b, 0) / ok.length
  const variance = ok.reduce((a, b) => a + (b - avg) ** 2, 0) / ok.length
  const diffs = ok.slice(1).map((v, i) => Math.abs(v - ok[i]!))
  return {
    sent,
    received: ok.length,
    lossPct: r1((sent - ok.length) / sent * 100),
    min: r1(sorted[0]!),
    avg: r1(avg),
    median: r1(percentile(sorted, 0.5)),
    p95: r1(percentile(sorted, 0.95)),
    max: r1(sorted.at(-1)!),
    stddev: r1(Math.sqrt(variance)),
    jitter: r1(diffs.length ? diffs.reduce((a, b) => a + b, 0) / diffs.length : 0),
  }
}

/** One timed request; null on error or timeout. Cache-busted so no intermediary can answer. */
export async function probe(url: string, signal?: AbortSignal, now = () => performance.now()): Promise<number | null> {
  const timeout = AbortSignal.timeout(REQUEST_TIMEOUT)
  const combined = signal ? AbortSignal.any([signal, timeout]) : timeout
  const start = now()
  try {
    const res = await fetch(`${url}${url.includes('?') ? '&' : '?'}_=${Date.now()}${Math.random().toString(36).slice(2, 6)}`, { cache: 'no-store', signal: combined, headers: { Accept: 'application/json' } })
    await res.arrayBuffer()
    return res.ok ? now() - start : null
  }
  catch (err) {
    if (signal?.aborted)
      throw err
    return null
  }
}

export function quality(s: Stats): 'excellent' | 'good' | 'fair' | 'poor' {
  if (s.lossPct > 5 || s.median > 300)
    return 'poor'
  if (s.lossPct > 1 || s.median > 150 || s.jitter > 50)
    return 'fair'
  return s.median > 50 || s.jitter > 20 ? 'good' : 'excellent'
}
