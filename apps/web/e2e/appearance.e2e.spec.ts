import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'

const ACCENTS = ['green', 'blue', 'indigo', 'purple', 'orange', 'red'] as const

test('sidebar footer shows version, copyright and licence below the tool list', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 })
  await page.goto('/')
  const footer = page.getByTestId('sidebar-footer')
  await expect(footer).toContainText(/NET-TOOLS v\d+\.\d+\.\d+/)
  await expect(footer).toContainText('© 2026 digibaro')
  await expect(footer.getByRole('link', { name: 'GPL-3.0-or-later' })).toHaveAttribute('href', /LICENSE$/)
  const menuBottom = await page.getByTestId('sidebar').locator('.n-menu').evaluate(el => el.getBoundingClientRect().bottom)
  const footerTop = await footer.evaluate(el => el.getBoundingClientRect().top)
  expect(footerTop).toBeGreaterThanOrEqual(menuBottom)
})

test('accent colour applies at once and is remembered', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'light' })
  await page.goto('/settings')
  const radio = page.getByTestId('accent-radio-purple')
  await radio.click()
  // Naive draws the checked radio dot in the primary colour.
  const dot = radio.locator('.n-radio__dot')
  await expect(dot).toHaveCSS('box-shadow', /rgb\(126, 48, 196\)/)
  await page.reload()
  await expect(page.getByTestId('accent-radio-purple').locator('input')).toBeChecked()
})

for (const accent of ACCENTS) {
  for (const scheme of ['light', 'dark'] as const) {
    test(`accent ${accent} has no axe violations in ${scheme} theme`, async ({ page }) => {
      await page.addInitScript(a => localStorage.setItem('nettools:accent', a), accent)
      await page.emulateMedia({ colorScheme: scheme })
      for (const path of ['/settings', '/dns-lookup?name=example.com']) {
        await page.goto(path)
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
        await page.waitForLoadState('networkidle')
        const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze()
        expect(results.violations.map(v => `${path} ${v.id}: ${v.nodes.map(n => n.target.join(' ')).join(', ')}`)).toEqual([])
      }
    })
  }
}
