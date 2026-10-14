import { expect, test } from '@playwright/test'

test('prefix-status: rejects private space', async ({ page }) => {
  await page.goto('/prefix-status?prefix=10.0.0.0/8')
  await expect(page.getByTestId('api-error')).toContainText('INVALID_INPUT')
})
