import type { Check } from './security-headers.service'
import { describe, expect, it } from 'vitest'
import { messageKey, sortChecks } from './security-headers.service'

const c = (id: string, status: Check['status'], detail = 'ok'): Check => ({ id, header: id, status, value: '', detail })

describe('security-headers', () => {
  it('sorts problems first', () => {
    expect(sortChecks([c('a', 'pass'), c('b', 'info'), c('c', 'fail'), c('d', 'warn')]).map(x => x.id)).toEqual(['c', 'd', 'a', 'b'])
  })
  it('builds message keys', () => {
    expect(messageKey(c('csp', 'warn', 'unsafe-inline'))).toBe('tools.security-headers.checks.csp.unsafe-inline')
    expect(messageKey(c('cookie', 'warn', 'HttpOnly,SameSite'))).toBe('tools.security-headers.checks.cookie.cookie')
  })
})
