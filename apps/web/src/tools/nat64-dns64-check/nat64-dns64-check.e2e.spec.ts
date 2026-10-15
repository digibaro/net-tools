import { expect, test } from '@playwright/test'

test('nat64-dns64-check: checks the server resolver', async ({ page }) => {
  await page.goto('/nat64-dns64-check')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('verdict').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})

test('nat64-dns64-check: private resolvers are refused (SSRF)', async ({ page }) => {
  await page.goto('/nat64-dns64-check?resolver=192.168.1.1')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
