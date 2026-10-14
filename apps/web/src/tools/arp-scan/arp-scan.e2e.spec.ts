import { expect, test } from '@playwright/test'

test('arp-scan: needs LAN mode', async ({ page }) => {
  await page.goto('/arp-scan')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('cidr')).first()).toBeVisible()
})
