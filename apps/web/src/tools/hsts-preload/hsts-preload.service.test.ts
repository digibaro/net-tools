import { describe, expect, it } from 'vitest'
import { statusType } from './hsts-preload.service'

describe('statusType', () => {
  it('maps statuses', () => {
    expect(statusType('preloaded')).toBe('success')
    expect(statusType('pending')).toBe('warning')
    expect(statusType('removed')).toBe('error')
    expect(statusType('unknown')).toBe('default')
  })
})
