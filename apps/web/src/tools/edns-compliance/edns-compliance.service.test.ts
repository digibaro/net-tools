import type { EdnsServer } from './edns-compliance.service'
import { describe, expect, it } from 'vitest'
import { grade } from './edns-compliance.service'

const s = (failed: string[], error = '') => ({ error, checks: ['dns', 'edns', 'edns1', 'ednsopt', 'tcp'].map(id => ({ id, pass: !failed.includes(id), detail: '' })) }) as EdnsServer

describe('edns-compliance', () => {
  it('grades a server', () => {
    expect(grade(s([]))).toBe('ok')
    expect(grade(s(['ednsopt']))).toBe('minor')
    expect(grade(s(['edns1']))).toBe('broken')
    expect(grade(s([], 'refused'))).toBe('unreachable')
  })
})
