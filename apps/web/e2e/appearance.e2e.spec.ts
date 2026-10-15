import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'

const ACCENTS = ['green', 'blue', 'indigo', 'purple', 'orange', 'red'] as const

test('sidebar shows the subtitle, and the footer version and licence below the tool list', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 })
  await page.goto('/')
  const footer = page.getByTestId('sidebar-footer')
  await expect(footer).toContainText(/NET-TOOLS v\d+\.\d+\.\d+/)
  await expect(footer).not.toContainText('©')
  await expect(page.getByTestId('app-subtitle')).toHaveText('Tools for Networkers')
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
      test.slow() // two full axe scans; slow when the suite starts many browsers at once
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

test('NL/UK switch translates the interface and is remembered', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 })
  await page.goto('/')
  await expect(page.getByTestId('locale-en')).toHaveAttribute('aria-pressed', 'true')
  await page.getByTestId('locale-nl').click()
  await expect(page.locator('html')).toHaveAttribute('lang', 'nl')
  await expect(page.getByTestId('app-subtitle')).toHaveText('Tools voor netwerkers')
  await expect(page.getByTestId('tool-card-ipv4-subnet-calculator')).toContainText('IPv4-subnetcalculator')
  await expect(page.getByTestId('sidebar')).toContainText('Connectiviteit & diagnose')
  await page.reload()
  await expect(page.getByTestId('locale-nl')).toHaveAttribute('aria-pressed', 'true')
  await expect(page.locator('html')).toHaveAttribute('lang', 'nl')
  // Search finds Dutch names.
  await page.keyboard.press('Control+k')
  await page.getByRole('combobox').fill('subnetcalculator')
  await expect(page.getByTestId('command-palette')).toContainText('IPv4-subnetcalculator')
  await page.keyboard.press('Escape')
  // A tool page in Dutch passes axe.
  await page.goto('/dns-lookup')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('DNS-lookup')
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze()
  expect(results.violations.map(v => v.id)).toEqual([])
  await page.getByTestId('locale-en').click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('DNS lookup')
})

for (const locale of ['en', 'nl'] as const) {
  test(`every header button is reachable at 360 px (${locale})`, async ({ page }) => {
    await page.addInitScript(l => localStorage.setItem('nettools:locale', l), locale)
    await page.setViewportSize({ width: 360, height: 740 })
    await page.goto('/')
    const header = page.locator('header').first()
    for (const button of await header.getByRole('button').all()) {
      const box = await button.boundingBox()
      expect(box, await button.getAttribute('aria-label') ?? 'button').not.toBeNull()
      expect(box!.x + box!.width, await button.getAttribute('aria-label') ?? 'button').toBeLessThanOrEqual(360)
    }
  })
}
