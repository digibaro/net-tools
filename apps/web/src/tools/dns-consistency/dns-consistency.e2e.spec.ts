import { expect, test } from '@playwright/test'

test('dns-consistency: requires a valid zone', async ({ page }) => {
  await page.goto('/dns-consistency')
  await page.getByTestId('zone').fill('not a zone')
  await expect(page.getByTestId('run')).toBeDisabled()
})
