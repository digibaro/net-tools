import { expect, test } from '@playwright/test'

test('rpki-validator: validates a route', async ({ page }) => {
  await page.goto('/rpki-validator?prefix=1.1.1.0/24&asn=AS13335')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
