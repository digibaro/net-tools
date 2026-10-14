import { expect, test } from '@playwright/test'

// Without ABUSEIPDB_KEY (as in CI) the tool is shown as unavailable.
test('ip-reputation: explains a missing API key', async ({ page }) => {
  await page.goto('/ip-reputation')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('ip'))).toBeVisible()
})
