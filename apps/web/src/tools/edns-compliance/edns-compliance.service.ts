import type { components } from '@/api/schema'

export type EdnsCompResult = components['schemas']['EdnsCompResult']
export type EdnsServer = EdnsCompResult['servers'][number]

export const CHECK_IDS = ['dns', 'edns', 'edns1', 'ednsopt', 'edns1opt', 'do', 'ednsflags', 'optlist', 'tcp'] as const

export type Grade = 'ok' | 'minor' | 'broken' | 'unreachable'

/** "broken" when plain EDNS or the version handling fails — resolvers then fall back or give up. */
export function grade(s: EdnsServer): Grade {
  if (s.error || s.checks.every(c => !c.pass))
    return 'unreachable'
  const failed = new Set(s.checks.filter(c => !c.pass).map(c => c.id))
  if (!failed.size)
    return 'ok'
  return ['dns', 'edns', 'edns1', 'do'].some(id => failed.has(id)) ? 'broken' : 'minor'
}
