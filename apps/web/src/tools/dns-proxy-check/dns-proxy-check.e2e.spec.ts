import { expect, test } from '@playwright/test'

test('dns-proxy-check: compares DNS and checks for proxies', async ({ page }) => {
  await page.goto('/dns-proxy-check')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('dns-verdict')).toBeVisible({ timeout: 30_000 })
  await expect(page.getByTestId('proxy-verdict').first()).toBeVisible()
})
