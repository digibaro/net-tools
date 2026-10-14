import { expect, test } from '@playwright/test'

test('dhcp-probe: needs LAN mode', async ({ page }) => {
  await page.goto('/dhcp-probe')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('interface')).first()).toBeVisible()
})
