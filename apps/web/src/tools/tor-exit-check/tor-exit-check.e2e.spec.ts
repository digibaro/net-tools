import { expect, test } from '@playwright/test'

test('tor-exit-check: checks an address', async ({ page }) => {
  await page.goto('/tor-exit-check?ip=8.8.8.8')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
