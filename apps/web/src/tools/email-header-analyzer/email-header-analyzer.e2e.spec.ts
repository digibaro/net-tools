import { expect, test } from '@playwright/test'

const HEADERS = `Received: from mx.example.net by inbox.example.net with ESMTPS; Tue, 13 Oct 2026 10:00:09 +0000
Received: from mail.example.com by mx.example.net with ESMTP; Tue, 13 Oct 2026 10:00:02 +0000
Authentication-Results: mx.example.net; spf=pass smtp.mailfrom=example.com; dkim=fail
Subject: Test`

test('email-header-analyzer: builds the hop timeline', async ({ page }) => {
  await page.goto('/email-header-analyzer')
  await page.getByTestId('input').locator('textarea').fill(HEADERS)
  await expect(page.getByTestId('result')).toContainText('7 s')
  await expect(page.getByTestId('auth')).toContainText('SPF pass')
})
