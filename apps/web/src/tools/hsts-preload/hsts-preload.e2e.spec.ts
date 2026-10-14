import { expect, test } from '@playwright/test'

test('hsts-preload: checks a domain', async ({ page }) => {
  await page.goto('/hsts-preload?domain=github.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
