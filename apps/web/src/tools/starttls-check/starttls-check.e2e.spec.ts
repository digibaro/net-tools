import { expect, test } from '@playwright/test'

test('starttls-check: defaults to SMTP and refuses private hosts', async ({ page }) => {
  await page.goto('/starttls-check?host=10.0.0.25')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
