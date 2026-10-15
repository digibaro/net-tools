import type { SmtpBannerResult } from './smtp-banner-check.service'
import { describe, expect, it } from 'vitest'
import { grade } from './smtp-banner-check.service'

const r = (failed: string[]) => ({ checks: ['ptr', 'fcrdns', 'banner', 'starttls'].map(id => ({ id, pass: !failed.includes(id), detail: '' })) }) as SmtpBannerResult

describe('smtp-banner-check', () => {
  it('grades the checks', () => {
    expect(grade(r([]))).toBe('good')
    expect(grade(r(['starttls']))).toBe('issues')
    expect(grade(r(['fcrdns']))).toBe('bad')
  })
})
