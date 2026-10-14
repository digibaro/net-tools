import { expect, test } from '@playwright/test'

test('wildcard-detector: checks a domain', async ({ page }) => {
  await page.goto('/wildcard-detector?domain=example.com')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
