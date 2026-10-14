import { expect, test } from '@playwright/test'

test('router-snippets: builds an IOS prefix-list and a Junos filter', async ({ page }) => {
  await page.goto('/router-snippets?list=192.0.2.0/24&name=TEST&upto4=26')
  await expect(page.getByTestId('output')).toHaveText('ip prefix-list TEST seq 5 permit 192.0.2.0/24 le 26')
  await page.goto('/router-snippets?list=192.0.2.0/24&name=TEST&kind=acl&platform=junos')
  await expect(page.getByTestId('output')).toContainText('set firewall family inet filter TEST term match from source-address 192.0.2.0/24')
})
