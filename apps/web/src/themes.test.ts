import { describe, expect, it } from 'vitest'
import { ACCENT_COLORS, ACCENTS, contrast, themeOverrides } from './themes'

// Surfaces from Naive UI 2.x: light = white; dark body / card / popover; dark primary-button text.
const LIGHT_SURFACES = ['#ffffff']
const DARK_SURFACES = ['#101014', '#18181c', '#48484e']
const DARK_BUTTON_TEXT = '#000000'

describe('accent presets', () => {
  it('matches the WCAG reference values', () => {
    expect(contrast('#000000', '#ffffff')).toBeCloseTo(21, 1)
    expect(contrast('#767676', '#ffffff')).toBeCloseTo(4.54, 1)
  })

  it.each(ACCENTS)('%s keeps >= 4.5:1 in light and dark mode', (accent) => {
    const { light, dark } = ACCENT_COLORS[accent]
    for (const bg of LIGHT_SURFACES) {
      expect(contrast(light.primary, bg), `${accent} light on ${bg}`).toBeGreaterThanOrEqual(4.5)
      expect(contrast(light.pressed, bg), `${accent} light pressed on ${bg}`).toBeGreaterThanOrEqual(4.5)
    }
    for (const bg of DARK_SURFACES)
      expect(contrast(dark.primary, bg), `${accent} dark on ${bg}`).toBeGreaterThanOrEqual(4.5)
    expect(contrast(dark.primary, DARK_BUTTON_TEXT), `${accent} dark button text`).toBeGreaterThanOrEqual(4.5)
  })

  it('builds Naive overrides and falls back to green', () => {
    expect(themeOverrides('blue', false).common?.primaryColor).toBe(ACCENT_COLORS.blue.light.primary)
    expect(themeOverrides('blue', true).Tag).toBeUndefined()
    expect(themeOverrides('nope' as never, true).common?.primaryColor).toBe(ACCENT_COLORS.green.dark.primary)
  })
})
