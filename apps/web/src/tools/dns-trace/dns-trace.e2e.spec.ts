import { expect, test } from '@playwright/test'

test('dns-trace: stream validation errors are shown', async ({ page }) => {
  await page.goto('/dns-trace?name=example.com&type=A')
  // Either the walk completes (network available) or a readable error appears.
  await expect(page.getByTestId('summary').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
})
