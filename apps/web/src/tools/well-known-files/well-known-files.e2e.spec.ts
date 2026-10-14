import { expect, test } from '@playwright/test'

test('well-known-files: checks a site', async ({ page }) => {
  await page.goto('/well-known-files?url=https://www.google.com/')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
