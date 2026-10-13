import { expect, test } from '@playwright/test'

test('whois-lookup: rejects invalid input', async ({ page }) => {
  await page.goto('/whois-lookup')
  await page.getByTestId('query').locator('input').fill('not valid')
  await expect(page.getByTestId('run')).toBeDisabled()
})

test('whois-lookup: queries a real domain', async ({ page }) => {
  await page.goto('/whois-lookup?query=example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
