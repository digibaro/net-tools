import { expect, test } from '@playwright/test'

test('dkim-checker: requires a selector', async ({ page }) => {
  await page.goto('/dkim-checker?domain=example.com')
  await expect(page.getByTestId('run')).toBeDisabled()
})
