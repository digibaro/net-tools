import { expect, test } from '@playwright/test'

test('wake-on-lan: needs LAN mode', async ({ page }) => {
  await page.goto('/wake-on-lan')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('mac')).first()).toBeVisible()
})
