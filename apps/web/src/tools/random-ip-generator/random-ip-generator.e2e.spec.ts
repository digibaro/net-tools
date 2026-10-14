import { expect, test } from '@playwright/test'

test('random-ip-generator: generates addresses in the prefix', async ({ page }) => {
  await page.goto('/random-ip-generator?prefix=192.0.2.0/30&count=2')
  await expect(page.getByTestId('result')).toContainText('192.0.2.')
  await page.goto('/random-ip-generator?prefix=192.0.2.0/30&count=3')
  await expect(page.getByTestId('no-result')).toBeVisible()
})
