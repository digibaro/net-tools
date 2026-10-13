import { expect, test } from '@playwright/test'

test('caa-checker: validates the name before querying', async ({ page }) => {
  await page.goto('/caa-checker')
  await page.getByTestId('name').fill('bad name')
  await expect(page.getByTestId('run')).toBeDisabled()
})
