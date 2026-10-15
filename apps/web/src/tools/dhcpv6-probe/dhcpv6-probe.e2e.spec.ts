import { expect, test } from '@playwright/test'

test('dhcpv6-probe: needs LAN mode', async ({ page }) => {
  await page.goto('/dhcpv6-probe')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('interface')).first()).toBeVisible()
})
