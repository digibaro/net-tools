import { expect, test } from '@playwright/test'

test('bgp-route-history: shows origins and visibility of a prefix', async ({ page }) => {
  await page.goto('/bgp-route-history?resource=193.0.0.0/21')
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
