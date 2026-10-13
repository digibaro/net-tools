import { expect, test } from '@playwright/test'

test('cidr-aggregator: merges prefixes from the URL', async ({ page }) => {
  await page.goto(`/cidr-aggregator?list=${encodeURIComponent('192.168.0.0/24\n192.168.1.0/24\nnope')}`)
  await expect(page.getByTestId('output')).toHaveText('192.168.0.0/23')
  await expect(page.getByText(/nope/)).toBeVisible()
})
