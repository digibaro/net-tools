import { expect, test } from '@playwright/test'

test('request-headers: shows the headers the server received', async ({ page }) => {
  await page.goto('/request-headers')
  const result = page.getByTestId('result')
  await expect(result.getByText('User-Agent', { exact: true })).toBeVisible()
  await expect(result.getByText('Added by NetTools').first()).toBeVisible()
})
