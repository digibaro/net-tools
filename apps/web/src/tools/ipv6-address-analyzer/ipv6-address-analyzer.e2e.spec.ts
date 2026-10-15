import { expect, test } from '@playwright/test'

test('ipv6-address-analyzer: derives the MAC from an EUI-64 address', async ({ page }) => {
  await page.goto('/ipv6-address-analyzer?ip=2001:db8::21b:21ff:fe3a:4b5c')
  await expect(page.getByTestId('result')).toContainText('00:1b:21:3a:4b:5c')
  await expect(page.getByTestId('result')).toContainText('ff02::1:ff3a:4b5c')
})
