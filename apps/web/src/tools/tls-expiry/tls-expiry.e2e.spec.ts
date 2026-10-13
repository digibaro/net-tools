import { expect, test } from '@playwright/test'

test('tls-expiry: private hosts are reported per row', async ({ page }) => {
  await page.goto('/tls-expiry?hosts=127.0.0.1')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('result')).toContainText('TARGET_PRIVATE_DENIED')
})
