import { expect, test } from '@playwright/test'

test('packet-decoder: decodes the sample DNS query', async ({ page }) => {
  await page.goto('/packet-decoder')
  await expect(page.getByTestId('layer')).toHaveCount(4)
  await expect(page.getByTestId('layer').last()).toContainText('example.com.')
})
