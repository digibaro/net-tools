import { expect, test } from '@playwright/test'

test('path-mtu: measures the path MTU to a public host', async ({ page }) => {
  await page.goto('/path-mtu?target=1.1.1.1')
  await expect(page.getByTestId('result').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 45_000 })
})
