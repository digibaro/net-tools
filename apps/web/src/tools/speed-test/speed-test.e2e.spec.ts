import { expect, test } from '@playwright/test'

// The e2e stack runs the public profile, where the speed test is disabled by default.
test('speed-test: is shown as disabled or runs', async ({ page }) => {
  await page.goto('/speed-test')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('run')).first()).toBeVisible()
})
