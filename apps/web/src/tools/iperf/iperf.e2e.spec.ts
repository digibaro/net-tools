import { expect, test } from '@playwright/test'

test('iperf: form renders (or binary_missing banner)', async ({ page }) => {
  await page.goto('/iperf')
  await expect(page.getByTestId('tool-disabled').or(page.getByTestId('server')).first()).toBeVisible()
})

test('iperf: private server refused in the public profile', async ({ page }) => {
  await page.goto('/iperf?server=10.0.0.1')
  const form = page.getByTestId('server')
  if (!(await form.isVisible()))
    test.skip(true, 'tool disabled in this stack')
  await expect(page.getByText(/private/i).first()).toBeVisible()
})
