import { expect, test } from '@playwright/test'

test('http-timing: times a public URL', async ({ page }) => {
  await page.goto(`/http-timing?url=${encodeURIComponent('https://example.com')}`)
  await expect(page.getByTestId('rating').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
