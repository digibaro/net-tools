import { expect, test } from '@playwright/test'

test('smtp-banner-check: checks reverse DNS of a mail server', async ({ page }) => {
  await page.goto('/smtp-banner-check?host=mx.nlnet.nl')
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})

test('smtp-banner-check: private hosts are refused (SSRF)', async ({ page }) => {
  await page.goto('/smtp-banner-check?host=10.0.0.25')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
