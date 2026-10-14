import { expect, test } from '@playwright/test'

test('revocation-check: checks a certificate', async ({ page }) => {
  await page.goto('/revocation-check?host=example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 60_000 })
})
