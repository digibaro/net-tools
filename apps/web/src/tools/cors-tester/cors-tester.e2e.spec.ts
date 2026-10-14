import { expect, test } from '@playwright/test'

test('cors-tester: validates the origin', async ({ page }) => {
  await page.goto('/cors-tester?url=https://example.com')
  await page.getByTestId('origin').fill('example.com/path')
  await expect(page.getByTestId('run')).toBeDisabled()
})
