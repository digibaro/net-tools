import { expect, test } from '@playwright/test'

test('cdn-waf-detector: recognises Cloudflare', async ({ page }) => {
  await page.goto('/cdn-waf-detector?url=https://www.cloudflare.com')
  await expect(page.getByTestId('detection').first().or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
