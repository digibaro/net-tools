import { expect, test } from '@playwright/test'

test('mac-address-lookup: resolves a vendor offline', async ({ page }) => {
  await page.goto('/mac-address-lookup?mac=00:0c:29:00:00:01')
  await expect(page.getByTestId('result').getByText('VMware, Inc.')).toBeVisible()
})
