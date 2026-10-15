import type { components } from '@/api/schema'

export type SmtpBannerResult = components['schemas']['SmtpBannerResult']

/** Checks that matter most for deliverability; a failure here gets mail rejected or junked. */
export const CRITICAL = ['ptr', 'fcrdns', 'ehloPtr'] as const

export function grade(r: SmtpBannerResult): 'good' | 'issues' | 'bad' {
  const failed = r.checks.filter(c => !c.pass).map(c => c.id)
  if (!failed.length)
    return 'good'
  return failed.some(id => (CRITICAL as readonly string[]).includes(id)) ? 'bad' : 'issues'
}
