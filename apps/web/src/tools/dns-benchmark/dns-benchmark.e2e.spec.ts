import { expect, test } from '@playwright/test'

test('dns-benchmark: refuses a private custom resolver', async ({ page }) => {
  await page.goto('/dns-benchmark?custom=10.0.0.53')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
