import { expect, test } from '@playwright/test'

test('punycode-converter: converts both ways', async ({ page }) => {
  await page.goto(`/punycode-converter?q=${encodeURIComponent('münchen.de')}`)
  await expect(page.getByTestId('result').getByText('xn--mnchen-3ya.de')).toBeVisible()
})
