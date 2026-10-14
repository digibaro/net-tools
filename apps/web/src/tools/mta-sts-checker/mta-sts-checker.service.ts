import type { components } from '@/api/schema'

export type EmailMtaSts = components['schemas']['EmailMtaSts']

/** "absent" when there is no record, else ok/warn/fail from the findings. */
export function sectionState(s: { record: string, problems: string[], warnings?: string[] }): 'absent' | 'ok' | 'warn' | 'fail' {
  if (!s.record)
    return 'absent'
  if (s.problems.length)
    return 'fail'
  return s.warnings?.length ? 'warn' : 'ok'
}
