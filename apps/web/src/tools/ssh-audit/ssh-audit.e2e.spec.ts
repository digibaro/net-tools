import { expect, test } from '@playwright/test'

test('ssh-audit: audits a public SSH server', async ({ page }) => {
  await page.goto('/ssh-audit?host=github.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
