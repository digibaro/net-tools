import { expect, test } from '@playwright/test'

test('ptr-coverage: scans a small public prefix', async ({ page }) => {
  await page.goto('/ptr-coverage?prefix=1.1.1.0/30')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})

test('ptr-coverage: too large prefixes are rejected inline', async ({ page }) => {
  await page.goto('/ptr-coverage?prefix=10.0.0.0/16')
  await expect(page.getByTestId('run')).toBeDisabled()
})
