import type { components } from '@/api/schema'

export type NtpCheck = components['schemas']['NtpCheck']
export { isValidHost } from '@/utils/host'

export const NTP_PRESETS = ['pool.ntp.org', 'time.cloudflare.com', 'time.google.com', 'time.apple.com', 'time.windows.com', 'ntp.ubuntu.com', 'ptbtime1.ptb.de']

/** Rough health rating of the server's offset relative to NetTools' own clock. */
export function offsetQuality(offsetMs: number): 'good' | 'fair' | 'poor' {
  const a = Math.abs(offsetMs)
  return a < 50 ? 'good' : a < 500 ? 'fair' : 'poor'
}
