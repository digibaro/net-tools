import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from '@/plugins/i18n.plugin'
import { fixed } from './format'
import { formatNumber } from './units'

describe('number formatting follows the UI language', () => {
  afterEach(() => {
    i18n.global.locale.value = 'en'
  })
  it('uses a decimal point in English and a comma in Dutch', () => {
    expect(fixed(12.345, 1)).toBe('12.3')
    expect(formatNumber(1234.5)).toBe('1,234.5')
    i18n.global.locale.value = 'nl'
    expect(fixed(12.345, 1)).toBe('12,3')
    expect(formatNumber(1234.5)).toBe('1.234,5')
  })
})
