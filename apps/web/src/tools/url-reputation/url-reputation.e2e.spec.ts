import { expect, test } from '@playwright/test'

// Needs at least one provider key; the e2e stack has none, so the disabled banner shows.
test('url-reputation: form or key-missing banner', async ({ page }) => {
  await page.goto('/url-reputation')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('url')).first()).toBeVisible()
})
