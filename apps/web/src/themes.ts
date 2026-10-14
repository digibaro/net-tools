import type { GlobalThemeOverrides } from 'naive-ui'

// Accents chosen for >= 4.5:1 contrast: #0f7a4a on white (5.4:1), #63e2b7 on #101014 (11.8:1).
export const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#0f7a4a',
    primaryColorHover: '#12905a',
    primaryColorPressed: '#0b5f39',
    primaryColorSuppl: '#12905a',
  },
  // Naive's default light tag text (e.g. #18a058 on its 10% tint) is ~3:1; darken to >= 5.7:1.
  Tag: {
    textColorSuccess: '#0b6b40',
    textColorInfo: '#1559b0',
    textColorWarning: '#8a5300',
    textColorError: '#b0203f',
  },
}

export const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#63e2b7',
    primaryColorHover: '#7fe7c4',
    primaryColorPressed: '#5acea7',
    primaryColorSuppl: '#2a947d',
  },
}

export const THEME_COLOR = { light: '#ffffff', dark: '#18181c' } as const
