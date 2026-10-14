import { expect, test } from '@playwright/test'

test('tls-scanner: refuses private hosts', async ({ page }) => {
  await page.goto('/tls-scanner?host=192.168.1.1')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
