import type { components } from '@/api/schema'

export type PortscanSummary = components['schemas']['PortscanSummary']
export interface OpenPort { port: number, service: string }

export const PRESETS = ['top100', 'web', 'mail', 'db', 'remote'] as const

/** Accepts a preset or "22,80,8000-8100" (the server enforces the port cap). */
export function isPortSpec(s: string): boolean {
  const v = s.trim().toLowerCase()
  if ((PRESETS as readonly string[]).includes(v))
    return true
  return v.split(',').every((part) => {
    const m = /^\s*(\d{1,5})(?:\s*-\s*(\d{1,5}))?\s*$/.exec(part)
    if (!m)
      return false
    const a = Number(m[1])
    const b = m[2] ? Number(m[2]) : a
    return a >= 1 && b <= 65535 && a <= b
  })
}
