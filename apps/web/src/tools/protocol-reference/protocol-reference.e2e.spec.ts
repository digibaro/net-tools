import { expect, test } from '@playwright/test'

test('protocol-reference: input from the URL produces a result', async ({ page }) => {
  await page.goto('/protocol-reference?q=x')
  await expect(page.getByTestId('result')).toHaveText('x')
})
