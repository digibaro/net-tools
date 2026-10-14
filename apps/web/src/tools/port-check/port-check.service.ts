import type { components } from '@/api/schema'

export type PortCheck = components['schemas']['PortCheck']
export { isValidHost } from '@/utils/host'

/** Mirrors the server rule: comma list and ranges, 1–65535, at most 10 ports. */
export function countPorts(input: string): number | undefined {
  const set = new Set<number>()
  for (const part of input.split(',').map(s => s.trim()).filter(Boolean)) {
    const m = /^(\d{1,5})(?:\s*-\s*(\d{1,5}))?$/.exec(part)
    if (!m)
      return undefined
    const a = Number(m[1])
    const b = Number(m[2] ?? m[1])
    if (a < 1 || b > 65535 || b < a || b - a > 10)
      return undefined
    for (let p = a; p <= b; p++)
      set.add(p)
  }
  return set.size || undefined
}

export const isValidPorts = (s: string) => (countPorts(s) ?? 0) > 0 && (countPorts(s) ?? 0) <= 10

export const STATE_TYPE: Record<string, 'success' | 'error' | 'warning' | 'default'> = {
  'open': 'success',
  'closed': 'error',
  'filtered': 'warning',
  'open|filtered': 'warning',
  'error': 'default',
}
