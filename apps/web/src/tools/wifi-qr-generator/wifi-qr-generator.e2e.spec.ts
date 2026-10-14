import { expect, test } from '@playwright/test'

test('wifi-qr-generator: renders a QR without putting the password in the URL', async ({ page }) => {
  await page.goto('/wifi-qr-generator?ssid=HomeNet')
  await page.getByTestId('password').fill('correct horse battery')
  await expect(page.getByTestId('qr')).toBeVisible()
  expect(page.url()).not.toContain('correct')
})
