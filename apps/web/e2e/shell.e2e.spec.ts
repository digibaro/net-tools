import { expect, test } from '@playwright/test'

test('Ctrl+K opens the command palette and runs a command', async ({ page }) => {
  await page.goto('/')
  await page.keyboard.press('Control+k')
  const palette = page.getByTestId('command-palette')
  await expect(palette).toBeVisible()
  await page.keyboard.type('about')
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL(/\/about$/)
})

test('palette theme commands change the theme', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'light' })
  await page.goto('/')
  await page.getByTestId('search-open').click()
  await page.keyboard.type('dark')
  await page.keyboard.press('Enter')
  await expect(page.locator('html')).toHaveClass(/dark/)
})

test('menu toggle hides and shows the sidebar on mobile', async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 740 })
  await page.goto('/')
  const sidebar = page.getByTestId('sidebar')
  await expect(sidebar).toHaveClass(/n-layout-sider--collapsed/)
  await expect(sidebar).toHaveAttribute('inert', '')
  await page.getByTestId('menu-toggle').click()
  await expect(sidebar).not.toHaveClass(/n-layout-sider--collapsed/)
  await expect(sidebar).not.toHaveAttribute('inert')
  expect((await sidebar.boundingBox())?.width).toBeGreaterThan(200)
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth)
  expect(overflow).toBe(false)
})
