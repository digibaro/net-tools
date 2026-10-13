import { expect, test } from '@playwright/test'

test('subnet-splitter: splits by prefix from the URL', async ({ page }) => {
  await page.goto('/subnet-splitter?net=10.0.0.0/24&mode=prefix&n=26')
  const result = page.getByTestId('result')
  await expect(result.getByRole('heading')).toHaveText('4 subnets of /26')
  await expect(result.getByText('10.0.0.192/26')).toBeVisible()
})
