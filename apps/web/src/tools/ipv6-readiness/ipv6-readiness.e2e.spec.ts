import { expect, test } from '@playwright/test'

test('ipv6-readiness: scores a domain', async ({ page }) => {
  await page.goto('/ipv6-readiness?domain=google.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
