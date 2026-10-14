import { expect, test } from '@playwright/test'

test('cookie-inspector: inspects a URL', async ({ page }) => {
  await page.goto('/cookie-inspector?url=https://www.google.com/')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
