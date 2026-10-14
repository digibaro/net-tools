import { expect, test } from '@playwright/test'

// OPEN_RELAY_TEST_ENABLED is off by default.
test('open-relay-test: opt-in', async ({ page }) => {
  await page.goto('/open-relay-test')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('host')).first()).toBeVisible()
})
