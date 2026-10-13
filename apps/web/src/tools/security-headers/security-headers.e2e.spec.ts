import { expect, test } from '@playwright/test'

test('security-headers: grades a public site', async ({ page }) => {
  await page.goto(`/security-headers?url=${encodeURIComponent('https://example.com')}`)
  await expect(page.getByTestId('grade').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
