import type { components } from '@/api/schema'

export type RelayResult = components['schemas']['RelayResult']
export type RelayCase = RelayResult['cases'][number]

export type Verdict = 'open' | 'closed' | 'inconclusive'

/** Open if any attempt was accepted; closed only when every attempt got an answer. */
export function verdict(r: RelayResult): Verdict {
  if (r.open)
    return 'open'
  if (r.error || r.cases.length === 0)
    return 'inconclusive'
  return 'closed'
}

/** "554 5.7.1 …" → 554 (0 when the step was not reached). */
export const replyCode = (reply: string) => Number(reply.slice(0, 3)) || 0
