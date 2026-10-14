import { expect, test } from '@playwright/test'

test('compression-check: checks a URL', async ({ page }) => {
  await page.goto('/compression-check?url=https://example.com/')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
