import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'

const isDark = () => document.documentElement.classList.contains('dark')

test('auto follows the OS scheme without stored preference', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'dark' })
  await page.goto('/')
  expect(await page.evaluate(isDark)).toBe(true)
})

test('auto flips live when the OS scheme changes', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'light' })
  await page.goto('/')
  await expect(page.locator('html')).not.toHaveClass(/dark/)
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(page.locator('html')).toHaveClass(/dark/)
})

test('explicit light survives reload while the OS is dark', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'dark' })
  await page.goto('/settings')
  await page.getByTestId('theme-radio-light').click()
  await expect(page.locator('html')).not.toHaveClass(/dark/)
  await page.reload()
  await expect(page.locator('html')).not.toHaveClass(/dark/)
  expect(await page.evaluate(() => localStorage.getItem('nettools:color-mode'))).toBe('light')
})

test('toggle cycles light → dark → auto', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'light' })
  await page.addInitScript(() => {
    if (!sessionStorage.getItem('seeded')) {
      localStorage.setItem('nettools:color-mode', 'light')
      sessionStorage.setItem('seeded', '1')
    }
  })
  await page.goto('/')
  const toggle = page.getByTestId('theme-toggle')
  await toggle.click()
  await expect(page.locator('html')).toHaveAttribute('data-color-mode', 'dark')
  await expect(page.locator('html')).toHaveClass(/dark/)
  await toggle.click()
  await expect(page.locator('html')).toHaveAttribute('data-color-mode', 'auto')
  await expect(page.locator('html')).not.toHaveClass(/dark/)
})

test('no flash: the theme class is set before the app script runs', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'light' })
  await page.addInitScript(() => {
    localStorage.setItem('nettools:color-mode', 'dark')
    document.addEventListener('DOMContentLoaded', () => {
      (window as unknown as { __darkAtDcl: boolean }).__darkAtDcl = document.documentElement.classList.contains('dark')
    })
  })
  // Hold the app bundle so only the inline script can have applied the class.
  await page.route('**/assets/index-*.js', async (route) => {
    await new Promise(r => setTimeout(r, 300))
    await route.continue()
  })
  await page.goto('/')
  expect(await page.evaluate(() => (window as unknown as { __darkAtDcl: boolean }).__darkAtDcl)).toBe(true)
})

for (const scheme of ['light', 'dark'] as const) {
  test(`home has no contrast violations in ${scheme} theme`, async ({ page }) => {
    await page.emulateMedia({ colorScheme: scheme })
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze()
    expect(results.violations.map(v => `${v.id}: ${v.nodes.map(n => n.target.join(' ')).join(', ')}`)).toEqual([])
  })
}
