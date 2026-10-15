import { expect, test } from '@playwright/test'

test('cache-header-analyzer: analyses a public URL', async ({ page }) => {
  await page.goto('/cache-header-analyzer?url=https://example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
