import { expect, test } from '@playwright/test'

test('reverse-dns: private addresses are refused in the public profile', async ({ page }) => {
  await page.goto('/reverse-dns?ip=10.1.2.3')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
