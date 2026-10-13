import { readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'

// Every tool folder is a route (docs/TOOL_AUTHORING.md); check the definition of done for all.
const toolsDir = join(import.meta.dirname, '../src/tools')
const tools = readdirSync(toolsDir).filter(d => statSync(join(toolsDir, d)).isDirectory())

for (const tool of tools) {
  test.describe(tool, () => {
    test('renders without console errors and fits 360px', async ({ page }) => {
      const errors: string[] = []
      page.on('console', (msg) => {
        if (msg.type() === 'error')
          errors.push(msg.text())
      })
      page.on('pageerror', err => errors.push(err.message))
      await page.setViewportSize({ width: 360, height: 740 })
      await page.goto(`/${tool}`)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      await page.waitForLoadState('networkidle')
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
      expect(errors).toEqual([])
    })

    for (const scheme of ['light', 'dark'] as const) {
      test(`has no axe violations in ${scheme} theme`, async ({ page }) => {
        await page.emulateMedia({ colorScheme: scheme })
        await page.goto(`/${tool}`)
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
        await page.waitForLoadState('networkidle')
        const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze()
        expect(results.violations.map(v => `${v.id}: ${v.nodes.map(n => n.target.join(' ')).join(', ')}`)).toEqual([])
      })
    }
  })
}
