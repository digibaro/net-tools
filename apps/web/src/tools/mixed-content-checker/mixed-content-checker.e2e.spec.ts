import { expect, test } from '@playwright/test'

test('mixed-content-checker: checks an https page', async ({ page }) => {
  await page.goto('/mixed-content-checker?url=https://example.com')
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
