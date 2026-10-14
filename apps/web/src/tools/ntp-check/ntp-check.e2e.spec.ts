import { expect, test } from '@playwright/test'

test('ntp-check: queries a public server', async ({ page }) => {
  await page.goto('/ntp-check?server=time.cloudflare.com')
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 15_000 })
})
