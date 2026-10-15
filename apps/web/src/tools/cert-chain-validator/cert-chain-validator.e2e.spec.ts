import { expect, test } from '@playwright/test'

test('cert-chain-validator: rejects input without certificates', async ({ page }) => {
  await page.goto('/cert-chain-validator')
  await page.getByTestId('pem').locator('textarea').fill('hello')
  await expect(page.getByTestId('run')).toBeDisabled()
})
