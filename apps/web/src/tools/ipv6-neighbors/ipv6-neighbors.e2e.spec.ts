import { expect, test } from '@playwright/test'

test('ipv6-neighbors: needs LAN mode', async ({ page }) => {
  await page.goto('/ipv6-neighbors')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('interface')).first()).toBeVisible()
})
