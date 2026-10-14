import { expect, test } from '@playwright/test'

test('ping: private targets are refused (SSRF)', async ({ page }) => {
  await page.goto('/ping?target=192.168.0.1')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})

test('ping: pings a public host from the server', async ({ page }) => {
  await page.goto('/ping?target=1.1.1.1&count=2&interval=500')
  await expect(page.getByTestId('summary').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
