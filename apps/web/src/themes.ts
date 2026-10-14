import type { GlobalThemeOverrides } from 'naive-ui'

/** Accent presets. Each pairs a dark shade (light mode) with a light tint (dark mode). */
export const ACCENTS = ['green', 'blue', 'indigo', 'purple', 'orange', 'red'] as const
export type Accent = typeof ACCENTS[number]
export const DEFAULT_ACCENT: Accent = 'green'

export interface Shades { primary: string, hover: string, pressed: string, suppl: string }

// Every pair is checked in themes.test.ts for >= 4.5:1 contrast: light shades against white (link
// text and white text on primary buttons), dark tints against the dark surfaces and against the
// dark text Naive puts on primary buttons.
export const ACCENT_COLORS: Record<Accent, { light: Shades, dark: Shades }> = {
  green: {
    light: { primary: '#0f7a4a', hover: '#12905a', pressed: '#0b5f39', suppl: '#12905a' },
    dark: { primary: '#63e2b7', hover: '#7fe7c4', pressed: '#5acea7', suppl: '#2a947d' },
  },
  blue: {
    light: { primary: '#1d5fc7', hover: '#2b6fd9', pressed: '#174b9e', suppl: '#2b6fd9' },
    dark: { primary: '#8cb8ff', hover: '#a6c8ff', pressed: '#7aa6ed', suppl: '#3d6fbf' },
  },
  indigo: {
    light: { primary: '#4f46e5', hover: '#6158ef', pressed: '#3f37c4', suppl: '#6158ef' },
    dark: { primary: '#a5b4fc', hover: '#bac6fd', pressed: '#93a2ea', suppl: '#5b61c9' },
  },
  purple: {
    light: { primary: '#7e30c4', hover: '#8f43d4', pressed: '#65259e', suppl: '#8f43d4' },
    dark: { primary: '#d4adfc', hover: '#e0c3fd', pressed: '#c299ea', suppl: '#8a52c4' },
  },
  orange: {
    light: { primary: '#b4490a', hover: '#c7560f', pressed: '#923a07', suppl: '#c7560f' },
    dark: { primary: '#fdb47a', hover: '#fdc596', pressed: '#eba267', suppl: '#c2702e' },
  },
  red: {
    light: { primary: '#b91c32', hover: '#cc2640', pressed: '#951527', suppl: '#cc2640' },
    dark: { primary: '#fda4af', hover: '#feb9c2', pressed: '#eb919d', suppl: '#c4505f' },
  },
}

// Naive's default light tag text (e.g. #18a058 on its 10% tint) is ~3:1; darken to >= 5.7:1.
const lightTagOverrides: GlobalThemeOverrides['Tag'] = {
  textColorSuccess: '#0b6b40',
  textColorInfo: '#1559b0',
  textColorWarning: '#8a5300',
  textColorError: '#b0203f',
}

export function themeOverrides(accent: Accent, dark: boolean): GlobalThemeOverrides {
  const c = (ACCENT_COLORS[accent] ?? ACCENT_COLORS[DEFAULT_ACCENT])[dark ? 'dark' : 'light']
  return {
    common: { primaryColor: c.primary, primaryColorHover: c.hover, primaryColorPressed: c.pressed, primaryColorSuppl: c.suppl },
    ...(dark ? {} : { Tag: lightTagOverrides }),
  }
}

/** WCAG 2.x contrast ratio of two #rrggbb colours. */
export function contrast(a: string, b: string): number {
  const lum = (hex: string) => {
    const [r, g, bl] = [1, 3, 5].map((i) => {
      const v = Number.parseInt(hex.slice(i, i + 2), 16) / 255
      return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4
    })
    return 0.2126 * r! + 0.7152 * g! + 0.0722 * bl!
  }
  const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x)
  return (hi! + 0.05) / (lo! + 0.05)
}

export const THEME_COLOR = { light: '#ffffff', dark: '#18181c' } as const
