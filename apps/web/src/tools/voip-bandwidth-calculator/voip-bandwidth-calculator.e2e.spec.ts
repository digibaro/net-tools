import { expect, test } from '@playwright/test'

test('voip-bandwidth-calculator: G.729 per call and Erlang B channels', async ({ page }) => {
  await page.goto('/voip-bandwidth-calculator?codec=G.729&link=ethernet-no-phy&bhca=600&avg=1&gos=1')
  await expect(page.getByTestId('result')).toContainText('31.2 kbit/s')
  await expect(page.getByTestId('erlang')).toContainText('18')
})
