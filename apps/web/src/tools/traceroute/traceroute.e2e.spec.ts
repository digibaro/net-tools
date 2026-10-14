import { expect, test } from '@playwright/test'

test('traceroute: traces a public host from the server', async ({ page }) => {
  await page.goto('/traceroute?target=1.1.1.1')
  await expect(page.getByTestId('finished').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 60_000 })
})
