import { expect, test } from '@playwright/test'

test('reverse-dns-name: PTR name from the URL', async ({ page }) => {
  await page.goto('/reverse-dns-name?q=192.0.2.10')
  await expect(page.getByTestId('result').getByText('10.2.0.192.in-addr.arpa')).toBeVisible()
})
