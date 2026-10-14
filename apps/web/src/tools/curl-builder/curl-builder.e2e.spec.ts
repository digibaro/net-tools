import { expect, test } from '@playwright/test'

test('curl-builder: parses a command into the builder', async ({ page }) => {
  await page.goto('/curl-builder')
  await page.getByTestId('paste').fill(`curl -X PUT https://api.example.com/x -H 'X-Test: 1' -d 'a=1'`)
  await expect(page.getByTestId('url')).toHaveValue('https://api.example.com/x')
  await expect(page.getByTestId('command')).toContainText('-X PUT')
  await expect(page.getByTestId('command')).toContainText(`-H 'X-Test: 1'`)
})
