import type { SshAudit } from './ssh-audit.service'
import { describe, expect, it } from 'vitest'
import { rating, severityOf } from './ssh-audit.service'

const a = (findings: SshAudit['findings']) => ({ findings }) as SshAudit

describe('ssh-audit', () => {
  it('rates servers', () => {
    expect(rating(a([]))).toBe('good')
    expect(rating(a([{ algorithm: 'ssh-rsa', kind: 'hostKey', severity: 'warn', reason: '' }]))).toBe('fair')
    expect(rating(a([{ algorithm: '3des-cbc', kind: 'cipher', severity: 'fail', reason: '' }]))).toBe('weak')
  })
  it('finds severities', () => {
    expect(severityOf(a([{ algorithm: 'x', kind: 'mac', severity: 'warn', reason: '' }]), 'x')).toBe('warn')
    expect(severityOf(a([]), 'x')).toBe('')
  })
})
