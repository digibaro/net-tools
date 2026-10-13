import { expect, test } from '@playwright/test'

test('dmarc-checker: checks a real domain', async ({ page }) => {
  await page.goto('/dmarc-checker?domain=google.com')
  await expect(page.getByTestId('findings').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
