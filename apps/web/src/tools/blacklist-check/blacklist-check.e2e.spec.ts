import { expect, test } from '@playwright/test'

test('blacklist-check: refuses private addresses', async ({ page }) => {
  await page.goto('/blacklist-check?target=10.0.0.1')
  await expect(page.getByTestId('api-error')).toContainText('INVALID_INPUT')
})
