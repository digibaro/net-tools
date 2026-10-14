import { expect, test } from '@playwright/test'

test('tls-inspector: inspects a public site', async ({ page }) => {
  await page.goto('/tls-inspector?host=example.com')
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
