import { expect, test } from '@playwright/test'

test('optical-power: converts and computes a budget', async ({ page }) => {
  await page.goto('/optical-power?p=0&u=dBm')
  await expect(page.getByTestId('converted')).toContainText('1 mW')
  await expect(page.getByTestId('verdict')).toBeVisible()
})
