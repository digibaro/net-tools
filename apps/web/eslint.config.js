import antfu from '@antfu/eslint-config'

export default antfu({
  vue: true,
  typescript: true,
  unocss: true,
  ignores: ['src/api/schema.d.ts', 'dist', 'playwright-report', 'test-results'],
}, {
  // eslint-plugin-unused-imports 4.4 misreports every variable in plain JS files as
  // "only used as a type" under ESLint 10; use the core rule there instead.
  files: ['**/*.{js,mjs,cjs}'],
  rules: {
    'unused-imports/no-unused-vars': 'off',
    'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
  },
})
