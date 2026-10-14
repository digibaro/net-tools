import type { components } from '@/api/schema'

export type SecurityHeaders = components['schemas']['SecurityHeaders']
export type Check = SecurityHeaders['checks'][number]

const ORDER: Record<Check['status'], number> = { fail: 0, warn: 1, pass: 2, info: 3 }

/** Problems first, then passes, then informational. */
export function sortChecks(checks: Check[]): Check[] {
  return [...checks].sort((a, b) => ORDER[a.status] - ORDER[b.status])
}

/** i18n key for a finding; falls back to the generic status text when not specific. */
export function messageKey(c: Check): string {
  return `tools.security-headers.checks.${c.id}.${c.detail.includes(',') ? 'cookie' : c.detail}`
}

export const GRADE_TYPE: Record<string, 'success' | 'warning' | 'error'> = { 'A+': 'success', 'A': 'success', 'B': 'warning', 'C': 'warning', 'D': 'error', 'F': 'error' }
