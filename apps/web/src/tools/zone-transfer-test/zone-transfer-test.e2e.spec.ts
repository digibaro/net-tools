import { expect, test } from '@playwright/test'

test('zone-transfer-test: tests a zone', async ({ page }) => {
  await page.goto('/zone-transfer-test?zone=example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
