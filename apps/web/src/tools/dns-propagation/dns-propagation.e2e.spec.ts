import { expect, test } from '@playwright/test'

test('dns-propagation: requires a valid name', async ({ page }) => {
  await page.goto('/dns-propagation')
  await page.getByTestId('name').fill('bad name')
  await expect(page.getByTestId('run')).toBeDisabled()
})
