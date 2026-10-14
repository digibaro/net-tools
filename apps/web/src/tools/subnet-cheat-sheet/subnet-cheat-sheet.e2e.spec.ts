import { expect, test } from '@playwright/test'

test('subnet-cheat-sheet: /26 row', async ({ page }) => {
  await page.goto('/subnet-cheat-sheet')
  await expect(page.getByTestId('v4').getByRole('row', { name: /^\/26 255\.255\.255\.192/ })).toContainText('62')
})
