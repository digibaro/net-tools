import { expect, test } from '@playwright/test'

test('mtr: private targets are refused', async ({ page }) => {
  await page.goto('/mtr?target=10.0.0.1')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
