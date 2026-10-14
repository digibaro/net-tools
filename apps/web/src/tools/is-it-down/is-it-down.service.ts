import type { RemoteProbeResult } from '@/composables/useRemote'
import { verdict } from '@/components/remote'

export type Overall = 'up' | 'partial' | 'down'

/** Overall status across probes: up everywhere, partly, or nowhere. */
export function overall(type: string, probes: RemoteProbeResult[]): Overall | undefined {
  if (!probes.length)
    return undefined
  const up = probes.filter(p => verdict(type, p.result) === 'up').length
  return up === probes.length ? 'up' : up === 0 ? 'down' : 'partial'
}

/** Accepts a host, an IP or a URL; returns the host to measure. */
export function hostOf(input: string): string {
  const s = input.trim()
  if (/^https?:\/\//i.test(s)) {
    try {
      return new URL(s).hostname.replace(/^\[|\]$/g, '')
    }
    catch {
      return s
    }
  }
  return s.replace(/\/.*$/, '')
}
