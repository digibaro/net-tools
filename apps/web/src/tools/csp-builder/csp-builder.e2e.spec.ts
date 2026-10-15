import { expect, test } from '@playwright/test'

test('csp-builder: evaluates a weak policy from the URL', async ({ page }) => {
  await page.goto(`/csp-builder?csp=${encodeURIComponent('script-src \'self\' \'unsafe-inline\'')}&output=nginx`)
  await expect(page.getByTestId('finding').first()).toContainText('\'unsafe-inline\'')
  await expect(page.getByTestId('output')).toContainText('add_header Content-Security-Policy')
})
