import { expect, test } from '@playwright/test'

test('irr-expansion: rejects option-like input', async ({ page }) => {
  await page.goto('/irr-expansion')
  const form = page.getByTestId('set')
  if (!(await form.isVisible()))
    test.skip(true, 'bgpq4 not installed in this stack')
  await form.fill('-h evil.example')
  await expect(page.getByTestId('run')).toBeDisabled()
})
