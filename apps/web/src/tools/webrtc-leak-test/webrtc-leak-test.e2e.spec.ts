import { expect, test } from '@playwright/test'

test('webrtc-leak-test: gathers candidates', async ({ page }) => {
  await page.goto('/webrtc-leak-test')
  await expect(page.getByTestId('result').first()).toBeVisible({ timeout: 30_000 })
})
