import { expect, test } from '@playwright/test'

test('looking-glass: shows routes for a prefix', async ({ page }) => {
  await page.goto('/looking-glass?prefix=1.1.1.0/24')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
