import { expect, test } from '@playwright/test'

test('edns-compliance: tests the nameservers of a zone', async ({ page }) => {
  await page.goto('/edns-compliance?zone=example.com')
  await expect(page.getByTestId('result').first().or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
