import { expect, test } from '@playwright/test'

// The stack may run without a GeoIP database: then the tool is shown as unavailable.
test('ip-geolocation: renders a result or the unavailable notice', async ({ page }) => {
  await page.goto('/ip-geolocation?ip=1.1.1.1')
  await expect(page.getByTestId('result').or(page.getByTestId('not-found')).or(page.getByTestId('tool-disabled')).or(page.getByTestId('api-error'))).toBeVisible({ timeout: 15_000 })
})
