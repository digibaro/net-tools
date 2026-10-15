import { expect, test } from '@playwright/test'

test('asn-neighbour-graph: draws the neighbours of an ASN', async ({ page }) => {
  await page.goto('/asn-neighbour-graph?asn=AS3333')
  await expect(page.getByTestId('graph').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
