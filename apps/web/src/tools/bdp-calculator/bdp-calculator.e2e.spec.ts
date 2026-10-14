import { expect, test } from '@playwright/test'

test('bdp-calculator: 1 Gbit/s × 50 ms', async ({ page }) => {
  await page.goto('/bdp-calculator?rate=1&ru=Gbps&rtt=50')
  await expect(page.getByTestId('bdp-result').getByTestId('kv-Bandwidth-delay product')).toHaveText(/^6\.25 MB/)
})
