import { expect, test } from '@playwright/test'

test('ip-special-check: flags CGNAT space as bogon', async ({ page }) => {
  await page.goto('/ip-special-check?ips=100.64.0.1')
  const result = page.getByTestId('result')
  await expect(result.getByText('Shared Address Space (CGNAT)')).toBeVisible()
  await expect(result.getByText('RFC 6598')).toBeVisible()
})
