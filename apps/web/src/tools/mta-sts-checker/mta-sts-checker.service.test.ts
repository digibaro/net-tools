import { describe, expect, it } from 'vitest'
import { sectionState } from './mta-sts-checker.service'

describe('sectionState', () => {
  it('classifies sections', () => {
    expect(sectionState({ record: '', problems: ['no-record'] })).toBe('absent')
    expect(sectionState({ record: 'v', problems: [] })).toBe('ok')
    expect(sectionState({ record: 'v', problems: [], warnings: ['mode-testing'] })).toBe('warn')
    expect(sectionState({ record: 'v', problems: ['x'] })).toBe('fail')
  })
})
