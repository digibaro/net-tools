import { expect, test } from '@playwright/test'

test('common-ixps: compares two networks', async ({ page }) => {
  await page.goto('/common-ixps?a=AS3333&b=AS13335')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
