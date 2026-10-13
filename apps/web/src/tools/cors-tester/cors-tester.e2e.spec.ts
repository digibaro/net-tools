import { expect, test } from '@playwright/test'

test('cors-tester: input from the URL produces a result', async ({ page }) => {
  await page.goto('/cors-tester?q=x')
  await expect(page.getByTestId('result')).toHaveText('x')
})
