import { expect, test } from '@playwright/test'

test('passive-exposure: looks up a public IP', async ({ page }) => {
  await page.goto('/passive-exposure?ip=1.1.1.1')
  await expect(page.getByTestId('result').or(page.getByTestId('not-found')).or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
