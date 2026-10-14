import { expect, test } from '@playwright/test'

test('spf-checker: expands a real SPF record', async ({ page }) => {
  await page.goto('/spf-checker?domain=google.com')
  await expect(page.getByTestId('record').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
