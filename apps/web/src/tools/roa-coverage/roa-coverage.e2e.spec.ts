import { expect, test } from '@playwright/test'

test('roa-coverage: RPKI status of the prefixes of an ASN', async ({ page }) => {
  await page.goto('/roa-coverage?asn=AS3333')
  await expect(page.getByTestId('coverage').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
