import { expect, test } from '@playwright/test'

test('asn-info: looks up an AS', async ({ page }) => {
  await page.goto('/asn-info?asn=AS13335')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
