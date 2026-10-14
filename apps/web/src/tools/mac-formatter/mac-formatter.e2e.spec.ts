import { expect, test } from '@playwright/test'

test('mac-formatter: formats from the URL and generates locally administered MACs', async ({ page }) => {
  await page.goto('/mac-formatter?mac=001a2b3c4d5e&format=cisco&upper=true')
  await expect(page.getByTestId('formatted')).toHaveText('001A.2B3C.4D5E')
  await page.getByTestId('generate').click()
  await expect(page.getByTestId('generated')).toHaveText(/^[0-9A-F][26AE][0-9A-F]{2}\./m)
})
