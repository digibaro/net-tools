import { expect, test } from '@playwright/test'

test('domain-expiry: checks a real domain', async ({ page }) => {
  await page.goto('/domain-expiry?domain=example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
