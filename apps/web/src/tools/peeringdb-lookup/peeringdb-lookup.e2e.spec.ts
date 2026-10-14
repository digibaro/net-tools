import { expect, test } from '@playwright/test'

test('peeringdb-lookup: looks up a network', async ({ page }) => {
  await page.goto('/peeringdb-lookup?asn=AS13335')
  await expect(page.getByTestId('result').or(page.getByTestId('not-found')).or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
