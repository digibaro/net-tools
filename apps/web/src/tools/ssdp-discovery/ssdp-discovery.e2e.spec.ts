import { expect, test } from '@playwright/test'

test('ssdp-discovery: needs LAN mode', async ({ page }) => {
  await page.goto('/ssdp-discovery')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('run')).first()).toBeVisible()
})
