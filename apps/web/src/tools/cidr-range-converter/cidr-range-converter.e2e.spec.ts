import { expect, test } from '@playwright/test'

test('cidr-range-converter: range from the URL becomes CIDRs', async ({ page }) => {
  await page.goto('/cidr-range-converter?q=10.0.0.0-10.0.2.255')
  const result = page.getByTestId('result')
  await expect(result.getByText('10.0.0.0/23')).toBeVisible()
  await expect(result.getByText('10.0.2.0/24')).toBeVisible()
})
