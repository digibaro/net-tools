import { expect, test } from '@playwright/test'

test('dns-record-types: input from the URL produces a result', async ({ page }) => {
  await page.goto('/dns-record-types?q=x')
  await expect(page.getByTestId('result')).toHaveText('x')
})
