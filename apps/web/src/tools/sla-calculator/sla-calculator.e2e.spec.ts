import { expect, test } from '@playwright/test'

test('sla-calculator: downtime per year for 99.9 %', async ({ page }) => {
  await page.goto('/sla-calculator?sla=99.9')
  await expect(page.getByTestId('result')).toContainText('8 h 45 min')
})
