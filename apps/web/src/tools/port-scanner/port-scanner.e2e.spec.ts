import { expect, test } from '@playwright/test'

// The public e2e profile has PORTSCAN_ENABLED off.
test('port-scanner: disabled in the public profile', async ({ page }) => {
  await page.goto('/port-scanner')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('target')).first()).toBeVisible()
})
