import { expect, test } from '@playwright/test'

test('rdap-lookup: queries a real IP', async ({ page }) => {
  await page.goto('/rdap-lookup?query=1.1.1.1')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
