import antfu from '@antfu/eslint-config'

export default antfu({
  vue: true,
  typescript: true,
  unocss: true,
  ignores: ['src/api/schema.d.ts', 'dist', 'playwright-report', 'test-results'],
})
