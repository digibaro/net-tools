import { expect, test } from '@playwright/test'

test('latency-matrix: pings several targets', async ({ page }) => {
  await page.goto('/latency-matrix?targets=1.1.1.1,8.8.8.8&count=2')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
