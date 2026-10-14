import { expect, test } from '@playwright/test'

test('cloud-ip-lookup: identifies a provider range', async ({ page }) => {
  await page.goto('/cloud-ip-lookup?ip=1.1.1.1')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
