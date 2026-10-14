import type { components } from '@/api/schema'

export type SshAudit = components['schemas']['SshAudit']

export function rating(a: SshAudit): 'good' | 'fair' | 'weak' {
  if (a.findings.some(f => f.severity === 'fail'))
    return 'weak'
  return a.findings.length ? 'fair' : 'good'
}

/** Severity of one algorithm, for highlighting it in the lists. */
export function severityOf(a: SshAudit, alg: string): 'fail' | 'warn' | '' {
  return a.findings.find(f => f.algorithm === alg)?.severity ?? ''
}
