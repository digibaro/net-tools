import { expect, test } from '@playwright/test'

test('http-timing: input from the URL produces a result', async ({ page }) => {
  await page.goto('/http-timing?q=x')
  await expect(page.getByTestId('result')).toHaveText('x')
})
