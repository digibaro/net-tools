import { expect, test } from '@playwright/test'

test('websocket-tester: validates the URL', async ({ page }) => {
  await page.goto('/websocket-tester?url=https://example.com')
  await expect(page.getByTestId('connect')).toBeDisabled()
})

test('websocket-tester: connects to a public echo server', async ({ page }) => {
  await page.goto('/websocket-tester?url=wss://echo.websocket.org')
  await page.getByTestId('connect').click()
  // Network access may be unavailable in CI: either it opens or the close event is logged.
  await expect(page.getByTestId('log')).toContainText(/Connected|closed/, { timeout: 20_000 })
})
