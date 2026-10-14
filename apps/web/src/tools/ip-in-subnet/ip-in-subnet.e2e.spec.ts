import { expect, test } from '@playwright/test'

test('ip-in-subnet: reports matches from the URL', async ({ page }) => {
  await page.goto('/ip-in-subnet?ips=10.9.9.9%0A1.1.1.1&prefixes=10.0.0.0/8')
  const result = page.getByTestId('result')
  await expect(result.getByRole('heading')).toHaveText('1 of 2 addresses matched')
  await expect(result.getByText('10.0.0.0/8')).toBeVisible()
})
