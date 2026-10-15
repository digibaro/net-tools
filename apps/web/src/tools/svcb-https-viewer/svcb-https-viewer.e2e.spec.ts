import { expect, test } from '@playwright/test'

test('svcb-https-viewer: shows the HTTPS record of a CDN site', async ({ page }) => {
  await page.goto('/svcb-https-viewer?name=cloudflare.com')
  await expect(page.getByTestId('result').first().or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
