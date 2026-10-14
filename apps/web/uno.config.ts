import { defineConfig, presetWind3 } from 'unocss'

export default defineConfig({
  presets: [presetWind3({ dark: 'class' })],
  shortcuts: {
    'nt-muted': 'text-neutral-600 dark:text-neutral-400',
  },
})
