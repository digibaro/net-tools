import { expect, test } from '@playwright/test'

test('data-unit-converter: Mbit/s to MB/s', async ({ page }) => {
  await page.goto('/data-unit-converter?kind=rate&value=100&unit=Mbps')
  const row = page.getByTestId('result').getByRole('row', { name: /^MB\/s/ })
  await expect(row).toContainText('12.5')
})
