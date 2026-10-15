import { i18n } from '@/plugins/i18n.plugin'

/** Number locale of the active UI language: decimal comma in Dutch, decimal point in English. */
export const numberLocale = () => (i18n.global.locale.value === 'nl' ? 'nl-NL' : 'en-GB')

/** Fixed decimals in the UI language: fixed(12.345, 1) → "12.3" (en) / "12,3" (nl). */
export function fixed(n: number, digits: number): string {
  return new Intl.NumberFormat(numberLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits }).format(n)
}
