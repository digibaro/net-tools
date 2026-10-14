import { expect, test } from '@playwright/test'

test('mdns-browser: needs LAN mode', async ({ page }) => {
  await page.goto('/mdns-browser')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('run')).first()).toBeVisible()
})
