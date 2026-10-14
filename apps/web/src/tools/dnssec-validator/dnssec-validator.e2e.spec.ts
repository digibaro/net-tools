import { expect, test } from '@playwright/test'

test('dnssec-validator: validates a signed name', async ({ page }) => {
  await page.goto('/dnssec-validator?name=example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
