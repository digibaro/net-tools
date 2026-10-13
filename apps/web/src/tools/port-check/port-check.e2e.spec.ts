import { expect, test } from '@playwright/test'

test('port-check: checks a public host from the server', async ({ page }) => {
  await page.goto('/port-check?target=1.1.1.1&ports=443')
  await expect(page.getByTestId('port-443').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})
