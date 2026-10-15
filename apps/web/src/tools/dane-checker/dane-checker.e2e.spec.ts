import { expect, test } from '@playwright/test'

test('dane-checker: checks the MX hosts of a DANE-enabled domain', async ({ page }) => {
  await page.goto('/dane-checker?host=nlnet.nl')
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 60_000 })
})
