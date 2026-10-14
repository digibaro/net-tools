import { expect, test } from '@playwright/test'

test('is-it-down: checks a site from probes worldwide', async ({ page }) => {
  await page.goto('/is-it-down?target=https://example.com/&probes=2')
  await expect(page.getByTestId('overall').or(page.getByTestId('api-error')).first()).toBeVisible({ timeout: 45_000 })
})
