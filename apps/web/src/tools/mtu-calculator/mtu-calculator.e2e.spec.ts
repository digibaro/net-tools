import { expect, test } from '@playwright/test'

test('mtu-calculator: PPPoE MSS', async ({ page }) => {
  await page.goto('/mtu-calculator?layers=pppoe')
  await expect(page.getByTestId('result').getByTestId('kv-TCP MSS (IPv4)')).toHaveText('1452')
  await page.getByTestId('layer-wireguard-v4').click()
  await expect(page.getByTestId('result').getByTestId('kv-Inner MTU')).toHaveText('1432')
})
