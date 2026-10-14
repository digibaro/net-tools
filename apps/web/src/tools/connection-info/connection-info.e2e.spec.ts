import { expect, test } from '@playwright/test'

test('connection-info: shows reachability and timings', async ({ page }) => {
  await page.goto('/connection-info')
  await expect(page.getByTestId('stack')).toBeVisible({ timeout: 15_000 })
  await expect(page.getByTestId('result')).toContainText('IPv4')
})
