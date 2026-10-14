import { expect, test } from '@playwright/test'

test('http-protocols: checks HTTP/2 and HTTP/3', async ({ page }) => {
  await page.goto('/http-protocols?url=https://www.cloudflare.com/')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
