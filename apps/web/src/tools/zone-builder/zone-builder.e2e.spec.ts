import { expect, test } from '@playwright/test'

test('zone-builder: adds a record to the zone', async ({ page }) => {
  await page.goto('/zone-builder?type=A')
  await page.getByTestId('name').locator('input').fill('www')
  await page.getByTestId('address').locator('input').fill('192.0.2.10')
  await page.getByTestId('add').click()
  await expect(page.getByTestId('zone')).toContainText('$ORIGIN example.com.')
  await expect(page.getByTestId('zone')).toContainText('192.0.2.10')
})
