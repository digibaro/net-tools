import { expect, test } from '@playwright/test'

test('lldp-neighbors: needs LAN mode', async ({ page }) => {
  await page.goto('/lldp-neighbors')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('interface')).first()).toBeVisible()
})
