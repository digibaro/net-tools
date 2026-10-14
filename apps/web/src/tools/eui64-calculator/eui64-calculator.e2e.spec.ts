import { expect, test } from '@playwright/test'

test('eui64-calculator: link-local and reverse MAC', async ({ page }) => {
  await page.goto('/eui64-calculator?mac=00:1a:2b:3c:4d:5e&addr=fe80::21a:2bff:fe3c:4d5e')
  await expect(page.getByTestId('result').getByTestId('kv-Link-local address')).toHaveText('fe80::21a:2bff:fe3c:4d5e')
  await expect(page.getByTestId('recovered').getByTestId('kv-MAC address')).toHaveText('00:1a:2b:3c:4d:5e')
})
