import { expect, test } from '@playwright/test'

test('browser-latency: measures round trips to the API', async ({ page }) => {
  await page.goto('/browser-latency?count=3&interval=250')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('quality')).toBeVisible({ timeout: 15_000 })
  await expect(page.getByTestId('result')).toContainText('3 / 3')
})
