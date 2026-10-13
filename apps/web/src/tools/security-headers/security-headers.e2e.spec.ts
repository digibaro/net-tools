import { expect, test } from '@playwright/test'

test('security-headers: input from the URL produces a result', async ({ page }) => {
  await page.goto('/security-headers?q=x')
  await expect(page.getByTestId('result')).toHaveText('x')
})
