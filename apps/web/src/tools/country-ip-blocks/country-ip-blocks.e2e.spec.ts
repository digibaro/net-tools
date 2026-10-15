import { expect, test } from '@playwright/test'

test('country-ip-blocks: lists the blocks of a country', async ({ page }) => {
  await page.goto('/country-ip-blocks?country=LU&family=4&format=cisco')
  await expect(page.getByTestId('country-name')).toContainText('Luxembourg')
  await expect(page.getByTestId('output').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 30_000 })
  if (await page.getByTestId('output').count())
    await expect(page.getByTestId('output').locator('textarea')).toHaveValue(/^ip prefix-list COUNTRY-LU permit /)
})
