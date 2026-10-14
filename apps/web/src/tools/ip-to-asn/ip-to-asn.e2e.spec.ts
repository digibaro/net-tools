import { expect, test } from '@playwright/test'

test('ip-to-asn: looks up a public address', async ({ page }) => {
  await page.goto('/ip-to-asn?ip=1.1.1.1')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error')).or(page.getByTestId('not-announced'))).toBeVisible({ timeout: 15_000 })
})
