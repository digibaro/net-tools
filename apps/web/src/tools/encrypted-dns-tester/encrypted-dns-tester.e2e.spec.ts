import { expect, test } from '@playwright/test'

test('encrypted-dns-tester: refuses private endpoints', async ({ page }) => {
  await page.goto('/encrypted-dns-tester?endpoint=192.168.1.1:853')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
