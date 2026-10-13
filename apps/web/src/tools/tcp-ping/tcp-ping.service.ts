export { isValidHost } from '@/utils/host'

/** Clamp a requested count/interval to the server limits. */
export function clampToLimits(count: number, intervalMs: number, limits?: { pingMaxCount: number, pingMinIntervalMs: number }) {
  const maxCount = limits?.pingMaxCount ?? 20
  const minInterval = limits?.pingMinIntervalMs ?? 200
  return {
    count: Math.min(Math.max(Math.round(count) || 1, 1), maxCount),
    interval: Math.min(Math.max(Math.round(intervalMs) || minInterval, minInterval), 5000),
  }
}
