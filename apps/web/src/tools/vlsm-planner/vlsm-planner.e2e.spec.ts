import { expect, test } from '@playwright/test'

test('vlsm-planner: plans the default example', async ({ page }) => {
  await page.goto('/vlsm-planner')
  await expect(page.getByTestId('result')).toContainText('192.168.1.0/25')
  await page.getByTestId('requirements').fill('A 300')
  await expect(page.getByTestId('no-result')).toBeVisible()
})
