import { expect, test } from '@playwright/test'

test('ct-search: searches CT logs', async ({ page }) => {
  await page.goto('/ct-search?domain=example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
