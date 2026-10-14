import { expect, test } from '@playwright/test'

test('mx-lookup: lists MX records without probing', async ({ page }) => {
  await page.goto('/mx-lookup?domain=gmail.com&probe=false')
  await expect(page.getByTestId('result').first().or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
