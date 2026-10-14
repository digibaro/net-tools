import { expect, test } from '@playwright/test'

test('gateway-info: needs LAN mode', async ({ page }) => {
  await page.goto('/gateway-info')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('run')).first()).toBeVisible()
})
