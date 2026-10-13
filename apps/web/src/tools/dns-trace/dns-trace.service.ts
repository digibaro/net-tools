import type { components } from '@/api/schema'

export type TraceStep = components['schemas']['DnsTraceStep']
export type TraceSummary = components['schemas']['DnsTraceSummary']

/** Text view of the walk, similar to `dig +trace`. */
export function traceText(steps: TraceStep[]): string {
  return steps.map((s) => {
    const head = `;; ${s.zone} via ${s.server} (${s.address}) ${s.rcode}${s.aa ? ' AA' : ''} in ${s.rttMs} ms`
    const body = s.answer.length
      ? s.answer.map(r => `${r.name}\t${r.ttl}\t${r.class}\t${r.type}\t${r.data}`)
      : s.referral.map(n => `  → ${n}`)
    return [head, ...body, ...(s.note ? [`;; ${s.note}`] : [])].join('\n')
  }).join('\n\n')
}
