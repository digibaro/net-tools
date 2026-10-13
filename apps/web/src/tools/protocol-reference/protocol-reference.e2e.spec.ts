import { expect, test } from '@playwright/test'

test('protocol-reference: ICMP type lookup', async ({ page }) => {
  await page.goto('/protocol-reference?tab=icmp&q=11')
  await expect(page.getByTestId('result').getByText('Time Exceeded')).toBeVisible()
})
