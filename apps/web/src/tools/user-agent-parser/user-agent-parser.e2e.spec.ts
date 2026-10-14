import { expect, test } from '@playwright/test'

test('user-agent-parser: defaults to the browser UA', async ({ page }) => {
  await page.goto('/user-agent-parser')
  await expect(page.getByTestId('input')).toHaveValue(/Mozilla\/5\.0/)
  await expect(page.getByTestId('result').getByTestId('kv-Browser')).toHaveText(/Chrom/)
})
