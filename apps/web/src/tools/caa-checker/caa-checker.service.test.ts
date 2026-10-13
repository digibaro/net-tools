import { describe, expect, it } from 'vitest'
import { issuerOf, verdict, wildcardCas } from './caa-checker.service'

describe('caa', () => {
  it('parses issuer domains', () => {
    expect(issuerOf('letsencrypt.org; validationmethods=dns-01')).toBe('letsencrypt.org')
    expect(issuerOf(';')).toBe('')
  })

  it('derives the verdict', () => {
    expect(verdict({ relevantName: '', issue: [], unknownCritical: [] })).toEqual({ kind: 'any' })
    expect(verdict({ relevantName: 'x', issue: ['letsencrypt.org', 'pki.goog; x=y'], unknownCritical: [] })).toEqual({ kind: 'restricted', cas: ['letsencrypt.org', 'pki.goog'] })
    expect(verdict({ relevantName: 'x', issue: [';'], unknownCritical: [] })).toEqual({ kind: 'blocked', reason: 'none-allowed' })
    expect(verdict({ relevantName: 'x', issue: ['a.org'], unknownCritical: ['tbs'] })).toEqual({ kind: 'blocked', reason: 'critical' })
    expect(verdict({ relevantName: 'x', issue: [], unknownCritical: [] })).toEqual({ kind: 'any' })
  })

  it('derives wildcard rules', () => {
    expect(wildcardCas({ relevantName: 'x', issue: ['a.org'], issueWild: [';'] })).toEqual([])
    expect(wildcardCas({ relevantName: 'x', issue: ['a.org'], issueWild: [] })).toEqual(['a.org'])
    expect(wildcardCas({ relevantName: '', issue: [], issueWild: [] })).toBe('any')
  })
})
