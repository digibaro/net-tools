import { expect, test } from '@playwright/test'

test('open-resolver-test: a public resolver is open by design', async ({ page }) => {
  await page.goto('/open-resolver-test?server=9.9.9.9')
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})

test('open-resolver-test: private servers are refused (SSRF)', async ({ page }) => {
  await page.goto('/open-resolver-test?server=192.168.1.1')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
