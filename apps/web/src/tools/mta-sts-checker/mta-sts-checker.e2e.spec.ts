import { expect, test } from '@playwright/test'

test('mta-sts-checker: checks a domain', async ({ page }) => {
  await page.goto('/mta-sts-checker?domain=gmail.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
